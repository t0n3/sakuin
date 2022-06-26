/*
Copyright © 2022 Tone

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/t0n3/sakuin/web"
)

type fileItem struct {
	Name  string
	Size  string
	Date  string
	IsDir bool
	Path  string
}

type templateVariables struct {
	Path  []breadcrumb
	Files []fileItem
}

type breadcrumb struct {
	Name string
	Path string
}

var dataDir string

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		dataDir = viper.GetString("data-dir")

		if dataDir == "" {
			log.Fatalln("Error: please specify a data directory, can't be empty")
		}

		_, err := os.Stat(dataDir)
		if err != nil {
			if os.IsNotExist(err) {
				log.Fatalln("Error: please specify a valid data directory")
			}
		}

		port := viper.GetInt("port")
		address := viper.GetString("listen-addr")

		mux := http.NewServeMux()
		mux.Handle("/assets/", web.AssetsHandler("/assets/", "dist"))
		mux.HandleFunc("/", serve)

		log.Printf("Starting Sakuin HTTP Server on %s:%d\n", address, port)
		http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), mux)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("data-dir", "d", "", "Directory containing data that Sakuin will serve")
	serveCmd.Flags().IntP("port", "p", 3000, "Port to listen to")
	serveCmd.Flags().String("listen-addr", "0.0.0.0", "Address to listen to")

	viper.BindPFlag("data-dir", serveCmd.Flags().Lookup("data-dir"))
	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("listen-addr", serveCmd.Flags().Lookup("listen-addr"))
}

func serve(w http.ResponseWriter, r *http.Request) {
	// Filepath, from the root data dir
	fp := filepath.Join(dataDir, filepath.Clean(r.URL.Path))
	// Cleaned filepath, without the root data dir, used for template rendering purpose
	cfp := strings.Replace(fp, dataDir, "", 1)

	// Return a 404 if the template doesn't exist
	info, err := os.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			notFound, _ := template.ParseFS(web.NotFound, "404.html")
			w.WriteHeader(http.StatusNotFound)
			notFound.ExecuteTemplate(w, "404.html", nil)
			log.Printf("404 - %s\n", cfp)
			return
		}
	}

	// Return a 404 if the request is for a directory
	if info.IsDir() {
		files, err := ioutil.ReadDir(fp)
		if err != nil {
			log.Fatal(err)
		}

		// Init template variables
		templateVars := templateVariables{}

		// Construct the breadcrumb
		path := strings.Split(cfp, "/")
		for len(path) > 1 {
			b := breadcrumb{
				Name: path[len(path)-1],
				Path: strings.Join(path, "/"),
			}
			path = path[:len(path)-1]
			templateVars.Path = append(templateVars.Path, b)
		}
		// Since the breadcrumb built is not very ordered...
		// REVERSE ALL THE THINGS
		for left, right := 0, len(templateVars.Path)-1; left < right; left, right = left+1, right-1 {
			templateVars.Path[left], templateVars.Path[right] = templateVars.Path[right], templateVars.Path[left]
		}

		// Establish list of files in the current directory
		for _, f := range files {
			templateVars.Files = append(templateVars.Files, fileItem{
				Name:  f.Name(),
				Size:  humanize.Bytes(uint64(f.Size())),
				Date:  humanize.Time(f.ModTime()),
				IsDir: f.IsDir(),
				Path:  filepath.Join(cfp, filepath.Clean(f.Name())),
			})
		}

		// Prepare the template
		tmpl, err := template.ParseFS(web.Index, "index.html")
		if err != nil {
			// Log the detailed error
			log.Println(err.Error())
			// Return a generic "Internal Server Error" message
			http.Error(w, http.StatusText(500), 500)
			return
		}

		// Return file listing in the template
		if err := tmpl.ExecuteTemplate(w, "index.html", templateVars); err != nil {
			log.Println(err.Error())
			http.Error(w, http.StatusText(500), 500)
		}
		log.Printf("200 - DIR %s\n", "/")
		return
	}

	if !info.IsDir() {
		content, _ := os.Open(fp)
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", info.Name()))
		http.ServeContent(w, r, fp, info.ModTime(), content)
		log.Printf("200 - FILE %s\n", cfp)
		return
	}
}
