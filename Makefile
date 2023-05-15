all: gnome-theme-changer install

gnome-theme-changer: gnome-theme-changer.go
	go build .

install: gnome-theme-changer
	sudo cp gnome-theme-changer /usr/local/bin/

