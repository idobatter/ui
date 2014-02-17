// +build !windows,!darwin,!plan9

// 16 february 2014
//package ui
package main

import (
	"fmt"
)

type sysData struct {
	cSysData

	widget		*gtkWidget
	container		*gtkWidget	// for moving
}

type classData struct {
	make	func() *gtkWidget
	makeAlt	func() *gtkWidget
	setText	func(widget *gtkWidget, text string)
	text		func(widget *gtkWidget) string
	append	func(widget *gtkWidget, text string)
	insert	func(widget *gtkWidget, index int, text string)
	selected	func(widget *gtkWidget) int
	// ...
	delete	func(widget *gtkWidget, index int)
	// ...
	signals	map[string]func(*sysData) func() bool
}

var classTypes = [nctypes]*classData{
	c_window:	&classData{
		make:	gtk_window_new,
		setText:	gtk_window_set_title,
		text:		gtk_window_get_title,
		signals:	map[string]func(*sysData) func() bool{
			"delete-event":		func(w *sysData) func() bool {
				return func() bool {
					if w.event != nil {
						w.event <- struct{}{}
					}
					return true		// do not close the window
				}
			},
			"configure-event":	func(w *sysData) func() bool {
				return func() bool {
					if w.container != nil && w.resize != nil {		// wait for init
						width, height := gtk_window_get_size(w.widget)
						// run in another goroutine since this will be called in uitask
						go func() {
							w.resize(0, 0, width, height)
						}()
					}
					// TODO really return true?
					return true		// do not continue events; we just did so
				}
			},
		},
	},
	c_button:		&classData{
		make:	gtk_button_new,
		setText:	gtk_button_set_label,
		text:		gtk_button_get_label,
		signals:	map[string]func(*sysData) func() bool{
			"clicked":		func(w *sysData) func() bool {
				return func() bool {
					if w.event != nil {
						w.event <- struct{}{}
					}
					return true		// do not close the window
				}
			},
		},
	},
	c_checkbox:	&classData{
		make:	gtk_check_button_new,
		setText:	gtk_button_set_label,
	},
	c_combobox:	&classData{
		make:	gtk_combo_box_text_new,
		// TODO creating an editable combobox causes GtkFixed to fail spectacularly for some reason
//		makeAlt:	gtk_combo_box_text_new_with_entry,
		makeAlt:	gtk_combo_box_text_new,
		// TODO setText
		text:		gtk_combo_box_text_get_active_text,
		append:	gtk_combo_box_text_append_text,
		insert:	gtk_combo_box_text_insert_text,
		selected:	gtk_combo_box_get_active,
		delete:	gtk_combo_box_text_remove,
	},
	c_lineedit:	&classData{
	},
	c_label:		&classData{
	},
	c_listbox:		&classData{
	},
}

func (s *sysData) make(initText string, window *sysData) error {
	ct := classTypes[s.ctype]
	if ct.make == nil {		// not yet implemented
		println(s.ctype, "not implemented")
		return nil
	}
	if s.alternate && ct.makeAlt == nil {		// not yet implemented
		println(s.ctype, "alt not implemented")
		return nil
	}
	ret := make(chan *gtkWidget)
	defer close(ret)
	uitask <- func() {
		if s.alternate {
			ret <- ct.makeAlt()
			return
		}
		ret <- ct.make()
	}
	s.widget = <-ret
	if window == nil {
		uitask <- func() {
			fixed := gtk_fixed_new()
			gtk_container_add(s.widget, fixed)
			// TODO return the container before assigning the signals?
			for signal, generator := range ct.signals {
				g_signal_connect(s.widget, signal, generator(s))
			}
			ret <- fixed
		}
		s.container = <-ret
	} else {
		s.container = window.container
		uitask <- func() {
			gtk_container_add(s.container, s.widget)
			for signal, generator := range ct.signals {
				g_signal_connect(s.widget, signal, generator(s))
			}
			ret <- nil
		}
		<-ret
	}
	err := s.setText(initText)
	if err != nil {
		return fmt.Errorf("error setting initial text of new window/control: %v", err)
	}
	return nil
}

func (s *sysData) show() error {
	ret := make(chan struct{})
	defer close(ret)
	uitask <- func() {
		gtk_widget_show(s.widget)
		ret <- struct{}{}
	}
	<-ret
	return nil
}

func (s *sysData) hide() error {
	ret := make(chan struct{})
	defer close(ret)
	uitask <- func() {
		gtk_widget_hide(s.widget)
		ret <- struct{}{}
	}
	<-ret
	return nil
}

func (s *sysData) setText(text string) error {
if classTypes[s.ctype] == nil || classTypes[s.ctype].setText == nil { return nil }
	ret := make(chan struct{})
	defer close(ret)
	uitask <- func() {
		classTypes[s.ctype].setText(s.widget, text)
		ret <- struct{}{}
	}
	<-ret
	return nil
}

func (s *sysData) setRect(x int, y int, width int, height int) error {
if classTypes[s.ctype] == nil || classTypes[s.ctype].make == nil { return nil }
	ret := make(chan struct{})
	defer close(ret)
	uitask <- func() {
		gtk_fixed_move(s.container, s.widget, x, y)
		gtk_widget_set_size_request(s.widget, width, height)
		ret <- struct{}{}
	}
	<-ret
	return nil
}

func (s *sysData) isChecked() bool {
	ret := make(chan bool)
	defer close(ret)
	uitask <- func() {
		ret <- gtk_toggle_button_get_active(s.widget)
	}
	return <-ret
}

func (s *sysData) text() string {
if classTypes[s.ctype] == nil || classTypes[s.ctype].make == nil { println(s.ctype,"unsupported text()"); return "" }
	ret := make(chan string)
	defer close(ret)
	uitask <- func() {
		ret <- classTypes[s.ctype].text(s.widget)
	}
	return <-ret
}

func (s *sysData) append(what string) error {
if classTypes[s.ctype] == nil || classTypes[s.ctype].make == nil { return nil }
	ret := make(chan struct{})
	defer close(ret)
	uitask <- func() {
		classTypes[s.ctype].append(s.widget, what)
		ret <- struct{}{}
	}
	<-ret
	return nil
}

func (s *sysData) insertBefore(what string, before int) error {
if classTypes[s.ctype] == nil || classTypes[s.ctype].make == nil { return nil }
	ret := make(chan struct{})
	defer close(ret)
	uitask <- func() {
		classTypes[s.ctype].insert(s.widget, before, what)
		ret <- struct{}{}
	}
	<-ret
	return nil
}

func (s *sysData) selectedIndex() int {
if classTypes[s.ctype] == nil || classTypes[s.ctype].make == nil { return -1 }
	ret := make(chan int)
	defer close(ret)
	uitask <- func() {
		ret <- classTypes[s.ctype].selected(s.widget)
	}
	return <-ret
}

func (s *sysData) selectedIndices() []int {
	// TODO
	return nil
}

func (s *sysData) selectedTexts() []string {
	// TODO
	return nil
}

func (s *sysData) setWindowSize(width int, height int) error {
	ret := make(chan struct{})
	defer close(ret)
	uitask <- func() {
		gtk_window_resize(s.widget, width, height)
		ret <- struct{}{}
	}
	<-ret
	return nil
}

func (s *sysData) delete(index int) error {
if classTypes[s.ctype] == nil || classTypes[s.ctype].make == nil { return nil }
	ret := make(chan struct{})
	defer close(ret)
	uitask <- func() {
		classTypes[s.ctype].delete(s.widget, index)
		ret <- struct{}{}
	}
	<-ret
	return nil
}