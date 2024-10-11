package data

type callbackListener struct {
	f func()
}

func (c *callbackListener) DataChanged() {
	c.f()
}
