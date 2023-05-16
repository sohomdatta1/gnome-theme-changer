# gnome-theme-changer

Theme changer for Gnome applications

## Disclaimer

> **Warning**
> Use this script at your own risk, This method of changing themes is at best a hack and is not supported by GNOME Developer community. The GNOME Foundation (and for that matter, anyone besides you) is/are not responsible for fixing any theming issues that may arise from using this tool.

## Usage

To use the script you, can run `gnome-theme-changer`, it will pop up a menu of themes that can be used both for gtk-3.0 and gtk-4.0.

If you are using it as part of a desktop rice, you can use `gnome-theme-changer s <THEME_NAME>`

To list all compatible themes, use `gnome-theme-changer l`

To get current theme, use `gnome-theme-changer c`

## Building from source

To build this script from source:

```sh
git clone https://github.com/sohomdatta1/gnome-theme-changer.git
```

And then run:

```sh
make -C gnome-theme-changer
```

## Additional info

This is a rewrite of [this python script](https://github.com/odziom91/libadwaita-theme-changer) by @odziom91.
