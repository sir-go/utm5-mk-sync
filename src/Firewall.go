package main

import (
	"fmt"
	"strings"
)

type ARec struct {
	Id       string // "*35AC"
	ListName string // "Deny"
	Address  string // "192.168.62.254"
	Comment  string // "mserg-hm#3780#5630"
	City     string // "tih"
}

type Firewall struct {
	device *MkDevice
	cache  []*ARec
}

func NewFirewall() *Firewall {
	log.Debug("init Firewall...")
	fw := &Firewall{device: &MkDevice{Cfg: cfg.Firewall}}
	eh(fw.device.Connect())
	fw.cache = make([]*ARec, 0)
	return fw
}

func (fw *Firewall) UpdateCache() {
	log.Debug("Update Firewall cache...")
	fw.cache = make([]*ARec, 0)

	log.Debugf("get ACL %s ...", cfg.Acl.ListAllow)
	eh(fw.device.AclGetAll(cfg.Acl.ListAllow, &fw.cache))
	total := len(fw.cache)
	log.Debugf("got %d records", total)

	log.Debugf("get ACL %s ...", cfg.Acl.ListDeny)
	eh(fw.device.AclGetAll(cfg.Acl.ListDeny, &fw.cache))
	log.Debugf("got %d records", len(fw.cache)-total)
}

func (fw *Firewall) Disconnect() {
	fw.device.Disconnect()
}

func (fw *Firewall) Add(rec *ARec) (err error) {
	if fw.FindByIp(rec.Address) != nil {
		return fmt.Errorf("fw already has ACL for ip %s", rec.Address)
	}
	if err = fw.device.AclAdd(rec); err != nil {
		return
	}
	fw.cache = append(fw.cache, rec)

	return
}

func (fw *Firewall) FindByComment(comment string) []*ARec {
	res := make([]*ARec, 0)
	for _, c := range fw.cache {
		if c.Comment == comment {
			res = append(res, c)
		}
	}
	if len(res) < 1 {
		log.Warningf("acl records with comment '%s' not found", comment)
	}
	return res
}

func (fw *Firewall) FindByCommentPartial(comment string) []*ARec {
	res := make([]*ARec, 0)
	for _, c := range fw.cache {
		lastDelimIndex := strings.LastIndex(c.Comment, "#")
		if lastDelimIndex > 0 && c.Comment[:lastDelimIndex] == comment {
			res = append(res, c)
		}
	}
	if len(res) < 1 {
		log.Warningf("acl records with comment part '%s' not found", comment)
	}
	return res
}

func (fw *Firewall) FindByIp(ip string) *ARec {
	for _, c := range fw.cache {
		if c.Address == ip {
			return c
		}
	}
	return nil
}

func (fw *Firewall) Del(rec *ARec) (err error) {
	log.Debug("- a: ", rec)
	for idx, c := range fw.cache {
		if rec.Id == c.Id {
			if err = fw.device.AclRemove(rec); err != nil {
				log.Errorf("couldn't remove ACL with comment '%s' and ip ",
					rec.Comment, rec.Id)
				return
			}
			if len(fw.cache)-1 == idx {
				fw.cache = fw.cache[:idx]
			} else {
				fw.cache = append(fw.cache[:idx], fw.cache[idx+1:]...)
			}
		}
	}
	return
}

func (fw *Firewall) Move(rec *ARec, toList string) (err error) {
	log.Debugf("move to address list '%s' record %s [%s]", toList, rec.ListName, rec.Address)
	if rec.ListName == toList {
		log.Warningf("%s %s already in the '%s' list", rec.Comment, rec.Address, toList)
		return nil
	}
	if err = fw.device.AclChange(rec, "list", toList); err == nil {
		rec.ListName = toList
	}
	return
}

func (fw *Firewall) Rename(rec *ARec, newName string) (err error) {
	log.Debugf("rename [%s] '%s' -> '%s'", rec.Address, rec.Comment, newName)
	if rec.Comment == newName {
		log.Warningf("'%s' == '%s' nothing to do", rec.ListName, newName)
		return nil
	}
	return fw.device.AclChange(rec, "comment", newName)
}

func (fw *Firewall) Sync(dbRec *DbRec, ip string) error {

	hRec := fw.FindByIp(ip)

	if hRec == nil {
		log.Noticef("%s: not found in Hashes -> add new one", ip)
		newRec := &ARec{
			ListName: cfg.Acl.ListDeny,
			Address:  ip,
			Comment:  dbRec.Name,
			City:     dbRec.CityCode,
		}
		if dbRec.Enabled {
			newRec.ListName = cfg.Acl.ListAllow
		}
		return fw.Add(newRec)
	}

	if hRec.City == "" {
		hRec.City = dbRec.CityCode
	}

	if dbRec.Name != hRec.Comment {
		log.Noticef("%s: eq IP but !eq Name (%s != %s) -> rename", ip, dbRec.Name, hRec.Comment)
		if err := fw.Rename(hRec, dbRec.Name); err != nil {
			return err
		}
	}

	if dbRec.Enabled && hRec.ListName != cfg.Acl.ListAllow {
		log.Noticef("%s: in %s but is Enabled -> move to %s", ip, cfg.Acl.ListDeny, cfg.Acl.ListAllow)
		return fw.Move(hRec, cfg.Acl.ListAllow)
	}

	if !dbRec.Enabled && hRec.ListName != cfg.Acl.ListDeny {
		log.Noticef("%s: in %s but is Disabled -> move to %s", ip, cfg.Acl.ListAllow, cfg.Acl.ListDeny)
		return fw.Move(hRec, cfg.Acl.ListDeny)
	}

	return nil
}

func (fw *Firewall) Clean() error {
	log.Debug("cleanup Firewall ...")
	toRemove := make([]*ARec, 0)
	for _, hRec := range fw.cache {
		if billing.FindByIP(hRec.Address) == nil {
			log.Noticef("%s: not found in Db -> remove", hRec.Address)
			toRemove = append(toRemove, hRec)
		}
	}
	for _, hRec := range toRemove {
		if hRec.City == "kor" {
			ppp.Kill(hRec.Address)
		}
		if err := fw.Del(hRec); err != nil {
			return err
		}
	}
	return nil
}
