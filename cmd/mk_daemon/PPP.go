package main

type PPPoE struct {
	devices []*MkDevice
}

func NewPPPoE() *PPPoE {
	LOG.Debug("init PPPoE...")
	p := &PPPoE{}
	p.devices = make([]*MkDevice, len(CFG.PPPoE))
	for idx, devConf := range CFG.PPPoE {
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
	LOG.Warningf("ip %s not found on PPPoE terminators", ip)
}
