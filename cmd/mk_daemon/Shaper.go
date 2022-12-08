package main

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

type QRec struct {
	Name    string    // "mserg-hm#3780"
	Target  []string  // ["192.168.62.254", "192.168.62.143"]
	Speed   int       // 42
	Comment string    // "PG" if Patriot
	Hash    string    // md5 of (Name + Speed + Comment + Target)
	Dev     *MkDevice // pointer to dev struct &{0x0000115}
}

func (qr *QRec) CalcSum() {
	h := fnv.New128()
	h.Write([]byte(qr.Name))
	h.Write([]byte(strconv.Itoa(qr.Speed)))
	h.Write([]byte(qr.Comment))
	for _, ip := range qr.Target {
		h.Write([]byte(ip))
	}
	qr.Hash = hex.EncodeToString(h.Sum(nil))
}

type Shaper struct {
	devices []*MkDevice
	cache   []*QRec
}

func NewShaper() *Shaper {
	LOG.Debug("init Shaper...")
	s := &Shaper{}
	s.devices = make([]*MkDevice, len(CFG.Shape.Devices))
	for idx, devConf := range CFG.Shape.Devices {
		s.devices[idx] = &MkDevice{devConf, nil, 0}
		eh(s.devices[idx].Connect())
	}
	s.cache = make([]*QRec, 0)
	return s
}

func (s *Shaper) Disconnect() {
	for _, d := range s.devices {
		d.Disconnect()
	}
}

func (s *Shaper) GetMinMaxUsageDevices() (minUDev, maxUDev *MkDevice) {
	minUDev = s.devices[0]
	maxUDev = s.devices[0]
	for _, d := range s.devices[1:] {
		if d.QueuesUsage < minUDev.QueuesUsage {
			minUDev = d
		}
		if d.QueuesUsage > maxUDev.QueuesUsage {
			maxUDev = d
		}
	}
	return
}

func (s *Shaper) UpdateCache() {
	LOG.Debug("Update shaper cache...")
	s.cache = make([]*QRec, 0)
	for _, mk := range s.devices {
		LOG.Debugf("get Queues Mk %s ...", mk.Cfg.Addr)

		queues := mk.QueueGetAll()
		LOG.Debugf("got %d records", len(queues))

		s.cache = append(s.cache, queues...)
	}
}

func (s *Shaper) Add(rec *QRec) (err error) {
	for _, r := range s.cache {
		if r.Name == rec.Name {
			return fmt.Errorf("this queue for %s already exists on device %s",
				rec.Name, r.Dev.Cfg.Addr)
		}
	}
	rec.Dev, _ = s.GetMinMaxUsageDevices()
	if err = rec.Dev.QueueAdd(rec); err != nil {
		return
	}

	s.cache = append(s.cache, rec)

	return
}

func (s *Shaper) Has(rec *QRec) *QRec {
	for _, c := range s.cache {
		if c.Hash == rec.Hash {
			return c
		}
	}
	return nil
}

func (s *Shaper) FindByIp(ip string) *QRec {
	for _, c := range s.cache {
		if IsInSlice(ip, c.Target) {
			return c
		}
	}
	return nil
}

func (s *Shaper) FindFirstByDev(dev *MkDevice) *QRec {
	for _, c := range s.cache {
		if c.Dev == dev && c.Dev != nil {
			return c
		}
	}
	return nil
}

func (s *Shaper) FindByName(name string) (rec *QRec) {
	for _, c := range s.cache {
		if c.Name == name {
			return c
		}
	}
	return
}

func (s *Shaper) Del(rec *QRec) (err error) {
	for idx, c := range s.cache {
		if rec.Hash == c.Hash {
			if err = c.Dev.QueueRemove(c); err != nil {
				LOG.Errorf("couldn't remove queue with name %s", rec.Name)
				return
			}
			if len(s.cache)-1 <= idx {
				s.cache = s.cache[:idx]
			} else {
				s.cache = append(s.cache[:idx], s.cache[idx+1:]...)
			}
		}
	}
	return
}

func (s *Shaper) Move(rec *QRec, dev *MkDevice) (err error) {
	if err = dev.QueueAdd(rec); err != nil {
		LOG.Errorf("couldn't add queue with name %s to dev %s", rec.Name, dev.Cfg.Addr)
		return
	}
	if err = rec.Dev.QueueRemove(rec); err != nil {
		LOG.Errorf("couldn't remove queue with name %s from dev %s", rec.Name, dev.Cfg.Addr)
		return
	}
	rec.Dev = dev
	return
}

func (s *Shaper) SetTarget(rec *QRec, ips []string) (err error) {
	if IsSlicesEqual(rec.Target, ips) {
		return fmt.Errorf("queue for %s targets are equal, nothing to do", rec.Name)
	}

	target := strings.Join(ips, ",")
	if err = rec.Dev.QueueSetTarget(rec, target); err != nil {
		return fmt.Errorf("can't change queue's '%s' target: %v", rec.Name, err)
	}
	rec.Target = ips[:]
	rec.CalcSum()
	return
}

func (s *Shaper) Sync(dbRec *DbRec) error {
	qRec := s.FindByName(dbRec.Name)

	if qRec != nil {
		if qRec.Hash == dbRec.Hash {
			return nil
		}
		LOG.Noticef("%s: eq Names but !eq Hashes -> remove hRec", dbRec.Name)
		if err := s.Del(qRec); err != nil {
			return err
		}
	}

	if dbRec.Speed > 0 {
		newQRec := &QRec{
			Name:    dbRec.Name,
			Target:  dbRec.Ips[:],
			Speed:   dbRec.Speed,
			Comment: dbRec.Comment,
		}
		newQRec.CalcSum()
		if err := s.Add(newQRec); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shaper) Clean() error {
	LOG.Debug("cleanup Shaper ...")
	toRemove := make([]*QRec, 0)
	for _, hRec := range s.cache {
		if dbRec := billing.FindByHash(hRec.Hash); dbRec == nil || dbRec.CityCode == "kor" {
			LOG.Noticef("%s: not found in Db -> remove", hRec.Name)
			toRemove = append(toRemove, hRec)
		}
	}
	for _, hRec := range toRemove {
		if err := s.Del(hRec); err != nil {
			return err
		}
	}
	return nil
}
