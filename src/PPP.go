package main

type PPPoE struct {
	devices []*MkDevice
}

func NewPPPoE() *PPPoE {
	log.Debug("init PPPoE...")
	p := &PPPoE{}
	p.devices = make([]*MkDevice, len(cfg.PPPoE))
	for idx, devConf := range cfg.PPPoE {
		p.devices[idx] = &MkDevice{Cfg: devConf}
		eh(p.devices[idx].Connect())
	}
	return p
}

func (p *PPPoE) Disconnect() {
	for _, p := range p.devices {
		p.Disconnect()
	}
}

func (p *PPPoE) Kill(ip string) {
	for _, d := range p.devices {
		if d.killPPPoE(ip) == nil {
			return
		}
	}
	log.Warningf("ip %s not found on PPPoE terminators", ip)
}
