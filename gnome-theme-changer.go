package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"
)

var THEME_PATH_TEMPLATES = []string{`/usr/share/themes`, `$HOME/.themes`, `$HOME/.local/share/themes`}

var LOCAL_CONFIG_PATH_TEMPLATE string = `$HOME/.config`

var REQUIRED_ASSETS = []string{"gtk-3.0/gtk.css", "gtk-3.0/gtk-dark.css", "gtk-4.0/gtk.css", "gtk-4.0/gtk-dark.css"}

var GTK_4_THEME_ASSETS = []string{"gtk-4.0/gtk.css", "gtk-4.0/gtk-dark.css", "gtk-4.0/assets"}

func fuzzyContains(a string, list []string) bool {
	for _, b := range list {
		if strings.Contains(a, b) {
			return true
		}
	}
	return false
}

func filter(list []fs.DirEntry, f func(fs.DirEntry) bool) []string {
	var result []string
	for _, s := range list {
		if f(s) {
			result = append(result, s.Name())
		}
	}
	return result
}

func IsGTKTheme(dir fs.DirEntry, baseDir string) bool {
	var c = 0
	err := filepath.WalkDir(path.Join(baseDir, dir.Name()), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("gnome-theme-changer: unable to access %s: %s", path, err)
		}

		if fuzzyContains(path, REQUIRED_ASSETS) {
			c++
		}
		return nil
	})

	if err != nil {
		fmt.Printf("gnome-theme-changer: unable to access %s: %s", dir.Name(), err)
	}

	if c >= len(REQUIRED_ASSETS) {
		return true
	}

	return false
}

func unionThemesLists(list_of_themes [][]string) (map[string]int, []string) {
	theme_map := make(map[string]int)
	for num, theme_list := range list_of_themes {
		for _, theme_name := range theme_list {
			theme_map[theme_name] = num
		}
	}

	theme_list := make([]string, 0)
	for theme_name := range theme_map {
		theme_list = append(theme_list, theme_name)
	}
	return theme_map, theme_list
}

func substEnvVar(env_var string, subst_string string) string {
	var env = os.Getenv(env_var)
	return strings.Replace(subst_string, `$`+env_var, env, -1)
}

func linkAllPartsOfTheme(theme_name string, base_dir string, theme_type string) {
	asset_dir := path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), theme_type)
	if _, err := os.Stat(asset_dir); os.IsNotExist(err) {
		err := os.Mkdir(asset_dir, 0755)
		if err != nil {
			fmt.Printf("gnome-theme-changer: unable to create gtk-4.0 directory: %v\n", err)
		}
	}

	all_assets, err := os.ReadDir(path.Join(base_dir, theme_name, theme_type))

	if err != nil {
		fmt.Printf("gnome-theme-changer: unable to access %s: %v", path.Join(base_dir, theme_name, theme_type), err)
	}

	for _, asset := range all_assets {
		var assets_path = path.Join(base_dir, theme_name, theme_type, asset.Name())
		var gtk_asset_path = path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), theme_type, asset.Name())
		if _, err := os.Stat(assets_path); os.IsNotExist(err) {
			return
		}
		err := os.Symlink(assets_path, gtk_asset_path)
		if err != nil {
			fmt.Printf("gnome-theme-changer: unable to link: %v\n", err)
		}
	}

	if _, err := os.Stat(path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), theme_type, `assets`)); os.IsNotExist(err) {
		err := os.Mkdir(path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), theme_type, `assets`), 0755)
		if err != nil {
			fmt.Printf("gnome-theme-changer: unable to create gtk-4.0 directory: %v\n", err)
		}
	}
}

func getGNOMETheme() string {
	theme_name, err := os.ReadFile(path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), "gtk-theme-name"))
	if err != nil {
		fmt.Printf("gnome-theme-changer: unable to read gtk-theme-name: %v\n", err)
	}
	return string(theme_name[:])
}

func setGNOMETheme(theme_name string, base_dir string) {
	linkAllPartsOfTheme(theme_name, base_dir, `gtk-4.0`)
	linkAllPartsOfTheme(theme_name, base_dir, `gtk-3.0`)
	gsettings_proc := exec.Command("gsettings", "set", "org.gnome.desktop.interface", "gtk-theme", theme_name)
	gsettings_proc.Run()
	gsettings_proc.Wait()
	os.WriteFile(path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), "gtk-theme-name"), []byte(theme_name), 0755)
}

func unsetGNOMETheme() {
	err := os.RemoveAll(path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), "gtk-4.0"))
	if err != nil {
		fmt.Printf("gnome-theme-changer: unable to remove gtk-4.0 directory: %v\n", err)
	}

	err = os.RemoveAll(path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), "gtk-3.0"))
	if err != nil {
		fmt.Printf("gnome-theme-changer: unable to remove gtk-4.0 directory: %v\n", err)
	}

	gsettings_proc := exec.Command("gsettings", "reset", "org.gnome.desktop.interface", "gtk-theme")
	gsettings_proc.Run()
	gsettings_proc.Wait()
	// https://gitlab.gnome.org/GNOME/libadwaita/-/blob/main/src/adw-style-manager.c#L258
	os.WriteFile(path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), "gtk-theme-name"), []byte("Adwaita-empty"), 0644)
}

func initializeThemeMapAndList() (map[string]int, []string) {
	list_of_themes := [][]string{}

	for _, theme_path_template := range THEME_PATH_TEMPLATES {
		theme_path := substEnvVar(`HOME`, theme_path_template)
		theme_entries, err := os.ReadDir(theme_path)

		if err != nil {
			continue
		}

		theme_entries_filtered := filter(theme_entries, func(theme_dir fs.DirEntry) bool {
			return IsGTKTheme(theme_dir, theme_path)
		})

		list_of_themes = append(list_of_themes, theme_entries_filtered)
	}

	theme_map, all_theme_entries := unionThemesLists(list_of_themes)

	sort.StringSlice(all_theme_entries).Sort()

	all_theme_entries = append(all_theme_entries, "Adwaita-empty")

	return theme_map, all_theme_entries

}

func MaybeSetGnomeTheme(all_theme_entries []string, theme_map map[string]int, result string) {
	index := slices.IndexFunc(all_theme_entries, func(elem string) bool { return elem == result })
	if result == `Adwaita-empty` {
		unsetGNOMETheme()
		return
	}

	if index == -1 {
		fmt.Println("Not a valid theme name")
		os.Exit(-1)
		return
	}

	if index == len(all_theme_entries)-1 {
		unsetGNOMETheme()
	} else {
		unsetGNOMETheme()
		setGNOMETheme(all_theme_entries[index], substEnvVar(`HOME`, THEME_PATH_TEMPLATES[theme_map[all_theme_entries[index]]]))
	}
}

func FirstRun() {
	first_run_cookie := path.Join(substEnvVar(`HOME`, LOCAL_CONFIG_PATH_TEMPLATE), "gnome-theme-changer")
	if _, err := os.Stat(first_run_cookie); os.IsNotExist(err) {
		fmt.Println(`
**WARNING** 
====================
This method of changing themes is at best a hack and is not supported by GNOME Developer community. 
The GNOME Foundation (and for that matter, anyone besides you) is/are not responsible for fixing 
any theming issues that may arise from using this tool.
		`)
		initial_prompt := promptui.Select{
			Label: `Do you want to continue`,
		}

		initial_prompt.Items = []string{"Yes", "No"}

		_, result, err := initial_prompt.Run()
		if result == "No" || err != nil {
			os.Exit(-1)
		}
		os.WriteFile(first_run_cookie, []byte("Yes"), 0755)
	}
}

func main() {
	FirstRun()
	app := cli.NewApp()
	app.Name = "gnome-theme-changer"
	app.Usage = "Change your GNOME theme"
	app.Version = "0.1.0"
	app.Authors = []*cli.Author{
		{Name: "Sohom Datta",
			Email: "sohomdatta1+gnome-theme-changer@gmail.com",
		},
		{
			Name:  "OdzioM",
			Email: "odziomek91@gmail.com",
		}}
	app.Action = func(c *cli.Context) error {

		theme_map, all_theme_entries := initializeThemeMapAndList()

		action := c.Args().Get(0)
		if action == "list-themes" || action == `l` {
			for _, theme := range all_theme_entries {
				fmt.Println(theme)
			}
			return nil
		} else if action == "current" || action == `c` {
			fmt.Println(getGNOMETheme())
			return nil
		} else if action == "set" || action == `s` {
			theme := c.Args().Get(1)
			if theme != "" {
				MaybeSetGnomeTheme(all_theme_entries, theme_map, theme)
			} else {
				fmt.Println("gnome-theme-changer: no theme specified")
			}
			return nil
		}

		theme_selection := promptui.Select{
			Label:             "Select theme",
			Items:             all_theme_entries,
			StartInSearchMode: true,
			Searcher: func(input string, index int) bool {
				return strings.Contains(all_theme_entries[index], input)
			},
		}

		_, result, err := theme_selection.Run()

		if err != nil {
			// Assume failure was intentional
			return nil
		}

		old_theme := getGNOMETheme()

		MaybeSetGnomeTheme(all_theme_entries, theme_map, result)

		fmt.Printf("Previewing %s theme\n", result)

		keep_using_new_theme := promptui.Select{
			Label: "Do you want to keep the changes",
			Items: []string{"Yes", "No"},
		}

		_, keep_using_new_theme_result, err := keep_using_new_theme.Run()

		if err != nil {
			MaybeSetGnomeTheme(all_theme_entries, theme_map, old_theme)
			// Assume failure was intentional
			return nil
		}

		if keep_using_new_theme_result == "Yes" {
			return nil
		}

		MaybeSetGnomeTheme(all_theme_entries, theme_map, old_theme)

		return nil
	}

	cli.AppHelpTemplate = `NAME:
	   {{.Name}} - {{.Usage}}
	USAGE:
	   {{.Name}} [command] [command options]

	VERSION:
	   {{.Version}}
	COMMANDS:
	   list-themes, l  List all available themes
	   current, c      Print the current theme
	   set, s          Set the current theme, requires a additional argument to specify the theme
	   help, h         Shows a list of commands or help for one command
	
	If the binary is run sans options, a prompt will be shown to select a theme.
	AUTHOR:
	   {{range .Authors}}{{ . }}{{end}}
	`

	err := app.Run(os.Args)

	if err != nil {
		fmt.Printf("gnome-theme-changer: unable to run app: %v\n", err)
	}

}
