package tcp

type Group struct {
	cores []core
	root  bool
	svr   *Engine
}

func (g *Group) Use(cores ...core) {
	g.cores = append(g.cores, cores...)
}

func (g *Group) TcpGroup() *Group {
	group := &Group{
		svr:   g.svr,
		root:  false,
		cores: nil,
	}
	if len(g.cores) > 0 {
		group.cores = append(group.cores, g.cores...)
	}
	return group
}

func (g *Group) Invoke(id int64, cores ...core) {
	var cs []core
	cs = append(cs, g.cores...)
	cs = append(cs, cores...)

	g.svr.addCore(id, cs...)
}
