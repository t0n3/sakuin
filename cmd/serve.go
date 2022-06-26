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
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/justinas/alice"
	"github.com/rs/zerolog/hlog"
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
	Run:   serve,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("data", "d", "", "Directory containing data that Sakuin will serve")
	serveCmd.Flags().IntP("port", "p", 3000, "Port to listen to")
	serveCmd.Flags().String("listen", "0.0.0.0", "Address to listen to")

	viper.BindPFlag("data", serveCmd.Flags().Lookup("data"))
	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("listen", serveCmd.Flags().Lookup("listen"))
}

func serve(cmd *cobra.Command, args []string) {
	dataDir = viper.GetString("data")

	if dataDir == "" {
		log.Fatal().Err(errors.New("please specify a data directory, can't be empty")).Msg("")
	}

	_, err := os.Stat(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatal().Err(errors.New("please specify a valid data directory")).Msg("")
		}
	}

	log.Info().Msgf("Sakuin will serve this directory: %s", dataDir)

	middleware := alice.New()

	// Install the logger handler with default output on the console
	middleware = middleware.Append(hlog.NewHandler(log))

	// Install some provided extra handler to set some request's context fields.
	// Thanks to that handler, all our logs will come with some prepopulated fields.
	middleware = middleware.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
	middleware = middleware.Append(hlog.RemoteAddrHandler("ip"))
	middleware = middleware.Append(hlog.UserAgentHandler("user_agent"))
	middleware = middleware.Append(hlog.RefererHandler("referer"))
	middleware = middleware.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

	handler := middleware.Then(http.HandlerFunc(serverHandler))
	assetsHandler := middleware.Then(web.AssetsHandler("/assets/", "dist"))

	port := viper.GetInt("port")
	address := viper.GetString("listen")

	mux := http.NewServeMux()
	mux.Handle("/assets/", assetsHandler)
	mux.Handle("/", handler)

	log.Info().Msgf("Starting Sakuin HTTP Server on %s:%d", address, port)
	http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), mux)
}

func serverHandler(w http.ResponseWriter, r *http.Request) {
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
			return
		}
	}

	// Return a 404 if the request is for a directory
	if info.IsDir() {
		files, err := ioutil.ReadDir(fp)
		if err != nil {
			log.Error().Err(err)
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
			log.Error().Err(err)
			// Return a generic "Internal Server Error" message
			http.Error(w, http.StatusText(500), 500)
			return
		}

		// Return file listing in the template
		if err := tmpl.ExecuteTemplate(w, "index.html", templateVars); err != nil {
			log.Error().Err(err)
			http.Error(w, http.StatusText(500), 500)
		}
		return
	}

	if !info.IsDir() {
		content, _ := os.Open(fp)
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", info.Name()))
		http.ServeContent(w, r, fp, info.ModTime(), content)
		return
	}
}
