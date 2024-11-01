package main

import "github.com/mattn/go-gtk"

func main() {
	gtk.Init(nil)
	w := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	w.SetTitle("Log Activity")
	vbox := gtk.NewVBox(false, 1)
	w.Add(vbox)
	w.ShowAll()
	gtk.Main()
}
