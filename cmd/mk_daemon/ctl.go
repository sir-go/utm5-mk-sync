package main

import (
	"encoding/json"
	"fmt"

	"github.com/streadway/amqp"
)

func Reject(err error, d *amqp.Delivery) {
	LOG.Error(err)
	if err = d.Reject(false); err != nil {
		LOG.Fatal(err)
	}
}

func Ack(d *amqp.Delivery) {
	if err := d.Ack(false); err != nil {
		LOG.Error(err)
	}
}

func MqMessagesHandler() {
	var err error
	for d := range mqMessages {
		t := new(Task)
		if err = json.Unmarshal(d.Body, t); err != nil {
			Reject(err, &d)
			continue
		}
		if err = HandleTask(t); err != nil {
			Reject(err, &d)
			continue
		}
		Ack(&d)
	}
}

func HandleTask(t *Task) error {
	LOG.Debug(t)
	switch t.Cmd {

	case "rebalance_q":
		shaper.UpdateCache()
		LOG.Info("rebalance queues ...")
		minDev, maxDev := shaper.GetMinMaxUsageDevices()
		diff := maxDev.QueuesUsage - minDev.QueuesUsage
		if diff < CFG.Shape.RebalanceThreshold {
			LOG.Infof("diff b/w max and min < %.2f, nothing to do", CFG.Shape.RebalanceThreshold)
			return nil
		}
		LOG.Debugf("diff b/w max and min == %.2f ...", diff)
		for diff > 0 {
			qRec := shaper.FindFirstByDev(maxDev)
			qRecInfo := *qRec
			LOG.Debugf("move queue %s -> %s ...", maxDev.Cfg.Addr, minDev.Cfg.Addr)
			ehSkip(shaper.Del(qRec))
			ehSkip(shaper.Add(&qRecInfo))
			diff -= float32(qRecInfo.Speed)
			LOG.Debugf("diff == %.2f now", diff)
		}

	case "sync_all":
		shaper.UpdateCache()
		fw.UpdateCache()
		billing.UpdateCache()
		LOG.Debug("sync with DB records ...")
		for _, u := range billing.cache {
			for _, ip := range u.Ips {
				ehSkip(fw.Sync(u, ip))
			}
			if u.CityCode != "kor" {
				ehSkip(shaper.Sync(u))
			}
		}
		LOG.Debug("sync is done.")
		LOG.Debug("cleanup ...")
		eh(fw.Clean())
		eh(shaper.Clean())
		LOG.Debug("cleanup is done.")
		return HandleTask(&Task{Cmd: "rebalance_q"})

	case "internet_on":
		for _, rec := range fw.FindByCommentPartial(t.User) {
			ehSkip(fw.Move(rec, CFG.Acl.ListAllow))
		}

	case "internet_off":
		for _, rec := range fw.FindByCommentPartial(t.User) {
			ehSkip(fw.Move(rec, CFG.Acl.ListDeny))
		}

	case "slink_add":
		ips := parseIps(t.Ips)
		if len(ips) == 0 {
			LOG.Warningf("no ip found: %s", t.Ips)
			return nil
		}

		if t.City != "kor" && !IsInSlice(t.User, CFG.Shape.IgnoreIds) {
			speed, comment, err := billing.GetTariffInfo(t.TlId)
			if err != nil {
				if err.Error() == "sql: no rows in result set" {
					LOG.Warningf("%s can't get tariff info for tlId: %d: %v", t.User, t.TlId, err)
					return nil
				}
				return err
			}

			if speed <= 0 {
				return nil
			}

			qRec := &QRec{
				Name:    t.User,
				Target:  ips,
				Speed:   speed,
				Comment: comment,
			}
			qRec.CalcSum()
			if err := shaper.Add(qRec); err != nil {
				return err
			}
		}

		for _, ip := range ips {
			aRec := &ARec{
				ListName: CFG.Acl.ListDeny,
				Address:  ip,
				Comment:  t.User,
				City:     t.City,
			}
			ehSkip(fw.Add(aRec))
		}

		return nil

	case "slink_del":
		aRecs := fw.FindByComment(t.User)
		if t.City == "kor" {
			for _, ar := range aRecs {
				LOG.Noticef("kor -> kill PPPoE for ip '%s'", ar.Address)
				ppp.Kill(ar.Address)
			}
		}
		for _, ar := range aRecs {
			ehSkip(fw.Del(ar))
		}
		if t.City != "kor" {
			if qRec := shaper.FindByName(t.User); qRec != nil {
				ehSkip(shaper.Del(qRec))
			}
		}

	case "slink_change":
		ips := parseIps(t.Ips)

		if len(ips) == 0 {
			LOG.Warningf("no ip found: %s", t.Ips)
			return nil
		}

		var (
			actualListName string
			actualIPs      []string
		)

		// remove address if not in new IPs
		for _, aRec := range fw.FindByComment(t.User) {
			actualIPs = append(actualIPs, aRec.Address)
			actualListName = aRec.ListName
			if !IsInSlice(aRec.Address, ips) {
				if t.City == "kor" {
					LOG.Noticef("kor -> kill PPPoE for ip '%s'", aRec.Address)
					ppp.Kill(aRec.Address)
				}
				ehSkip(fw.Del(aRec))
			}
		}

		if len(actualIPs) == 0 || actualListName == "" {
			LOG.Warningf("no addresses for '%s' found in hashes", t.User)
			return nil
		}

		// add new address if not in actual
		for _, ip := range ips {
			if IsInSlice(ip, actualIPs) {
				continue
			}
			newAddress := &ARec{
				ListName: actualListName,
				Address:  ip,
				Comment:  t.User,
				City:     t.City,
			}
			ehSkip(fw.Add(newAddress))
		}

		if t.City == "kor" || IsInSlice(t.User, CFG.Shape.IgnoreIds) {
			return nil
		}

		qRec := shaper.FindByName(t.User)
		if qRec == nil {
			return fmt.Errorf("queue for user %s not found", t.User)
		}
		return shaper.SetTarget(qRec, ips)

	}
	return nil
}
