package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"math"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/bamiaux/rez"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/gorilla/mux"

	"image/draw"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/crypto/ssh/terminal"
	_ "golang.org/x/image/webp"
)

var configPath string
var filePointers FilePointerList
var bleveIndex bleve.Index

func main() {
	var err error

	flag.StringVar(&configPath, "c", "./config/sugoi.json", "Path to the configuration file. Default: ./config/sugoi.json")
	user := flag.Bool("u", false, "Adds a new user or changes de password of an existing user on your config file.")
	flag.Parse()

	InitializeBuildTime()
	err = InitializeConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if user != nil && *user {
		ManageUsers()
		return
	}

	InitializeSession()

	err = InitializeFilePointers()
	if err != nil {
		fmt.Println(err)
		os.Exit(5)
	}

	err = InitializeBleve()
	if err != nil {
		fmt.Println(err)
		os.Exit(8)
	}

	InitializeOrder()

	r := mux.NewRouter()

	r.HandleFunc("/files.json", func(w http.ResponseWriter, r *http.Request) {
		if _, ret := HandleAuth(w, r); ret {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filePointers.List)
		return
	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, ret := HandleAuth(w, r); ret {
			return
		}

		data := struct {
			Title          string
			Things         []*Thing
			HasNext        bool
			HasPrev        bool
			Page           int
			PageNextUrl    string
			PagePrevUrl    string
			Format         string
			OrderAvailable OrderFields
			Order          string
			Search         string
		}{
			Title:          "Home",
			HasNext:        false,
			HasPrev:        false,
			OrderAvailable: orderFields,
		}
		titleBuilder := strings.Builder{}

		data.Page, _ = strconv.Atoi(r.FormValue("page"))

		if data.Page < 0 {
			data.Page = 0
		}

		data.Order = r.FormValue("order")

		if _, ok := orderFields.Find(data.Order); !ok {
			data.Order = "CreatedAt"
		}
		data.Search = r.FormValue("q")

		fFormat := r.FormValue("format")
		var pageSize int
		switch fFormat {
		case "table":
			data.Format = "table"
			pageSize = 50

		// case "covers":
		default:
			data.Format = "covers"
			pageSize = 48
		}

		fQ := strings.TrimSpace(r.FormValue("q"))
		fDebug := r.FormValue("debug")

		var query query.Query
		if len(fQ) == 0 {
			titleBuilder.WriteString("Home")
			query = bleve.NewMatchAllQuery()
		} else {
			rawquery := bleve.NewQueryStringQuery(fQ)
			parsed, err := rawquery.Parse()

			if err != nil {
				RenderError(w, r, err.Error())
				return
			}

			r := strings.NewReplacer(
				"+", "",
				"\"", "",
			)
			titleBuilder.WriteString(r.Replace(fQ))

			query = parsed
		}

		if data.Page >= 1 {
			titleBuilder.WriteString(" - Page ")
			titleBuilder.WriteString(strconv.Itoa(data.Page))
		}

		data.Title = titleBuilder.String()

		search := bleve.NewSearchRequest(query)
		search.Size = pageSize + 1
		search.From = data.Page * pageSize
		search.Fields = []string{"*"}

		search.SortBy([]string{data.Order})

		if fDebug == "query" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(search)
			return
		}

		searchResults, err := bleveIndex.Search(search)
		if err != nil {
			RenderError(w, r, err.Error())
			return
		}

		if fDebug == "raw" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(searchResults)
			return
		}

		for _, v := range searchResults.Hits {
			thing, err := NewThingFromHash(v.ID)
			if err != nil {
				RenderError(w, r, err.Error())
				return
			}
			data.Things = append(data.Things, thing)
		}

		if len(data.Things) > pageSize {
			data.HasNext = true
			data.Things = data.Things[:pageSize]
			q := r.URL.Query()
			q.Set("page", strconv.Itoa(data.Page+1))
			u := url.URL{Path: r.URL.Path, RawQuery: q.Encode()}
			data.PageNextUrl = u.String()
		}

		if data.Page > 0 {
			data.HasPrev = true
			q := r.URL.Query()
			q.Set("page", strconv.Itoa(data.Page-1))
			u := url.URL{Path: r.URL.Path, RawQuery: q.Encode()}
			data.PagePrevUrl = u.String()
		}

		RenderPage(w, r, "index.gohtml", data)
	})

	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
			Error string
		}{
			Title: "Login",
		}

		fUser := r.FormValue("user")
		fPassword := r.FormValue("password")

		if r.Method == http.MethodPost {
			found := false
			foundUser := ""
			foundPass := ""
			userLowercase := strings.ToLower(fUser)

			for k, v := range config.Users {
				if userLowercase == strings.ToLower(k) {
					foundUser = k
					foundPass = v
					found = found || true
				}
			}

			if !found {
				data.Error = "User not found"
				RenderPage(w, r, "login.gohtml", data)
				return
			}

			if !CheckPasswordHash(fPassword, foundPass) {
				data.Error = "Wrong or empty password"
				RenderPage(w, r, "login.gohtml", data)
				return
			}

			session, _ := sessionStore.Get(r, config.SessionCookieName)
			session.Values["authenticated"] = true
			session.Values["user"] = foundUser
			err := session.Save(r, w)

			if err != nil {
				log.Println(err)
			}

			returnPage := r.FormValue("return")
			if len(returnPage) > 0 && returnPage[0] == '/' {
				http.Redirect(w, r, returnPage, 302)
				return
			}

			http.Redirect(w, r, "/", 302)
			return
		}

		RenderPage(w, r, "login.gohtml", data)
	})

	r.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			session, _ := sessionStore.Get(r, config.SessionCookieName)
			session.Values["authenticated"] = false
			session.Values["user"] = nil
			session.Save(r, w)
		}

		http.Redirect(w, r, "/login", 302)
	})

	r.HandleFunc("/thing/details/{hash:[a-z0-9]+}.json", func(w http.ResponseWriter, r *http.Request) {
		if _, ret := HandleAuth(w, r); ret {
			return
		}

		vars := mux.Vars(r)
		vHash, _ := vars["hash"]

		thing, err := NewThingFromHash(vHash)
		if err != nil {
			RenderError(w, r, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(thing)
		return
	})

	r.HandleFunc("/thing/details/{hash:[a-z0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		if _, ret := HandleAuth(w, r); ret {
			return
		}

		vars := mux.Vars(r)
		vHash, _ := vars["hash"]

		thing, err := NewThingFromHash(vHash)
		if err != nil {
			RenderError(w, r, err.Error())
			return
		}

		data := struct {
			Title    string
			Thing    *Thing
			FilesRaw []string
		}{
			Title: thing.Title,
			Thing: thing,
		}

		data.FilesRaw, _ = thing.ListFilesRaw()
		RenderPage(w, r, "thingDetails.gohtml", data)
	})

	r.HandleFunc("/thing/{hash:[a-z0-9]+}/rating.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, ret := CheckAuth(w, r)
		if ret {
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(JsonResponse{"Unauthorized"})
			return
		}

		vars := mux.Vars(r)
		vHash := vars["hash"]

		thing, err := NewThingFromHash(vHash)
		if err != nil {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(JsonError{err.Error()})
			return
		}

		if r.Method == http.MethodPost {
			fRate, err := strconv.Atoi(r.FormValue("rate"))
			fToggle, _ := strconv.ParseBool(r.FormValue("toggle"))

			if err != nil {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(JsonError{err.Error()})
				return
			}

			if fRate > 5 {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(JsonResponse{"Rating cannot be > 5"})
				return
			}

			if fRate < 0 {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(JsonResponse{"Rating cannot be < 0"})
				return
			}

			if fToggle && thing.Rating == fRate {
				err = thing.TrySaveRating(0)
			} else {
				err = thing.TrySaveRating(fRate)
			}

			if err != nil {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(JsonError{err.Error()})
				return
			}

			type Response struct {
				JsonResponse
				Rating int
			}
			var response Response
			response.Rating = thing.Rating

			response.Message = fmt.Sprintf("Rating updated to %d", thing.Rating)

			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(struct{ Rating int }{thing.Rating})
			return
		}

		w.WriteHeader(405)
		json.NewEncoder(w).Encode(JsonResponse{"method not allowed"})
	})

	r.HandleFunc("/thing/{hash:[a-z0-9]+}/cover.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, ret := CheckAuth(w, r)
		if ret {
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(JsonResponse{"Unauthorized"})
			return
		}

		vars := mux.Vars(r)
		vHash := vars["hash"]

		thing, err := NewThingFromHash(vHash)
		if err != nil {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(JsonError{err.Error()})
			return
		}

		if r.Method == http.MethodPost {
			fFile := r.FormValue("file")

			if err != nil {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(JsonError{err.Error()})
				return
			}

			changed := (thing.Cover != fFile)
			if changed {
				err := thing.TrySaveCover(fFile, false)
				if err != nil {
					w.WriteHeader(400)
					json.NewEncoder(w).Encode(JsonError{err.Error()})
					return
				}
			}

			type Response struct {
				JsonResponse
				Cover string
			}
			var response Response
			response.Cover = thing.Cover

			if changed {
				response.Message = fmt.Sprintf("Cover updated to %s", thing.Cover)
			} else {
				response.Message = "This is the cover already"
			}

			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(struct{ Cover string }{thing.Cover})
			return
		}

		w.WriteHeader(405)
		json.NewEncoder(w).Encode(JsonResponse{"method not allowed"})
	})

	r.HandleFunc("/thing/{hash:[a-z0-9]+}/addMark.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, ret := CheckAuth(w, r)
		if ret {
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(JsonResponse{"Unauthorized"})
			return
		}

		vars := mux.Vars(r)
		vHash := vars["hash"]

		thing, err := NewThingFromHash(vHash)
		if err != nil {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(JsonError{err.Error()})
			return
		}

		if r.Method == http.MethodPost {
			err = thing.AddMark()

			if err != nil {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(JsonError{err.Error()})
				return
			}

			type Response struct {
				JsonResponse
				Rating int
			}
			var response Response
			response.Rating = thing.Rating

			response.Message = fmt.Sprintf("Marks: %d", thing.Marks)

			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(struct{ Rating int }{thing.Rating})
			return
		}

		w.WriteHeader(405)
		json.NewEncoder(w).Encode(JsonResponse{"method not allowed"})
	})

	r.HandleFunc("/thing/{hash:[a-z0-9]+}/subMark.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, ret := CheckAuth(w, r)
		if ret {
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(JsonResponse{"Unauthorized"})
			return
		}

		vars := mux.Vars(r)
		vHash := vars["hash"]

		thing, err := NewThingFromHash(vHash)
		if err != nil {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(JsonError{err.Error()})
			return
		}

		if r.Method == http.MethodPost {
			err = thing.SubMark()

			if err != nil {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(JsonError{err.Error()})
				return
			}

			type Response struct {
				JsonResponse
				Rating int
			}
			var response Response
			response.Rating = thing.Rating

			response.Message = fmt.Sprintf("Marks: %d", thing.Marks)

			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(struct{ Rating int }{thing.Rating})
			return
		}

		w.WriteHeader(405)
		json.NewEncoder(w).Encode(JsonResponse{"method not allowed"})
	})

	r.HandleFunc("/thing/read/{hash:[a-z0-9]+}{page:/?[0-9]*}", func(w http.ResponseWriter, r *http.Request) {
		if _, ret := HandleAuth(w, r); ret {
			return
		}

		vars := mux.Vars(r)
		vHash, _ := vars["hash"]
		vPage, _ := strconv.Atoi(strings.TrimLeft(vars["page"], "/"))

		thing, err := NewThingFromHash(vHash)
		if err != nil {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(JsonError{err.Error()})
			return
		}

		data := struct {
			Title string
			Thing *Thing
			Files []string
			Page  int
			Hash  string
		}{
			Title: thing.Title,
			Thing: thing,
			Page:  vPage,
			Hash:  vHash,
		}

		data.Files, err = thing.ListFiles()
		if err != nil {
			RenderError(w, r, err.Error())
			return
		}
		RenderPage(w, r, "thingRead.gohtml", data)
	})

	r.HandleFunc("/thing/file/{hash:[a-z0-9]+}/{file:.+}", func(w http.ResponseWriter, r *http.Request) {
		if _, ret := HandleAuth(w, r); ret {
			return
		}

		vars := mux.Vars(r)
		vHash, _ := vars["hash"]
		vFile, _ := vars["file"]
		fSize := r.FormValue("size")

		if fSize == "thumb" {
			if RenderThumbCache(w, r, vHash, vFile) {
				return
			}
		}

		thing, err := NewThingFromHash(vHash)
		if err != nil {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(JsonError{err.Error()})
			return
		}

		reader, closers, err := thing.getFileReader(vFile)
		if err != nil {
			RenderError(w, r, err.Error())
			return
		}

		defer MultiClose(closers)

		wantedMime := mime.TypeByExtension(filepath.Ext(vFile))

		if fSize == "thumb" {
			original, _, err := image.Decode(reader)
			if err != nil {
				RenderError(w, r, err.Error())
				return
			}

			oriRect := original.Bounds()
			factor := float64(oriRect.Max.X) / 256.0
			dstHeight := float64(oriRect.Max.Y) / factor

			dstRect := image.Rect(0, 0, 256, int(math.Ceil(dstHeight)))

			in := image.NewRGBA(oriRect)
			draw.Draw(in, oriRect, original, oriRect.Min, draw.Src)
			original = nil

			out := image.NewRGBA(dstRect)
			err = rez.Convert(out, in, rez.NewBicubicFilter())
			if err != nil {
				RenderError(w, r, err.Error())
				return
			}
			in = nil

			w.Header().Set("Content-Type", "image/jpeg")
			SetCacheHeader(w, r, 31536000)

			jpegOptions := jpeg.Options{
				Quality: 60,
			}

			if config.CacheThumbnails {
				cacheWriter := ThumbCacheTarget(vHash, vFile)
				if cacheWriter != nil {
					jpeg.Encode(cacheWriter, out, &jpegOptions)
					cacheWriter.Close()
				}
			}

			err = jpeg.Encode(w, out, &jpegOptions)
			out = nil
			if err != nil {
				return
			}
		} else {
			w.Header().Set("Content-Type", wantedMime)
			SetCacheHeader(w, r, 31536000)
			_, err = io.Copy(w, reader)
			if err != nil {
				RenderError(w, r, err.Error())
				return
			}
		}
	})

	var reindexJob ReindexJob
	r.HandleFunc("/system", func(w http.ResponseWriter, r *http.Request) {
		if _, ret := HandleAuth(w, r); ret {
			return
		}

		var err error
		r.ParseForm()

		if r.Method == http.MethodPost {
			fAction := r.FormValue("action")

			switch fAction {
			case "reindex":
				w.Header().Set("Content-Type", "application/json")
				if reindexJob.Running {
					json.NewEncoder(w).Encode(reindexJob)
					return
				}

				err = reindexJob.Start()
				if err != nil {
					json.NewEncoder(w).Encode(JsonError{err.Error()})
					return
				}

				http.Redirect(w, r, "/system", 302)
				// http.Redirect(w, r, "/system?action=reindexStatus", 302)
				return

			case "cancelReindex":
				reindexJob.RequestCancel = true

				http.Redirect(w, r, "/system?action=reindexStatus", 302)
				return

			case "reload":
				w.Header().Set("Content-Type", "application/json")
				prev := len(filePointers.List)
				err = InitializeFilePointers()
				if err != nil {
					w.WriteHeader(500)
					json.NewEncoder(w).Encode(JsonError{err.Error()})
					return
				}

				count := len(filePointers.List)
				json.NewEncoder(w).Encode(JsonResponse{fmt.Sprintf("%d files found (previously %d files)", count, prev)})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(JsonError{fmt.Sprintf("Unknown operation '%s'", fAction)})
			return
		}

		fAction := r.FormValue("action")

		switch fAction {
		case "reindexStatus":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(reindexJob)
			return
		}

		data := struct {
			Title   string
			Reindex ReindexJob
		}{
			Title:   "System",
			Reindex: reindexJob,
		}

		RenderPage(w, r, "system.gohtml", data)
	})

	r.HandleFunc("/allFiles", func(w http.ResponseWriter, r *http.Request) {
		if _, ret := HandleAuth(w, r); ret {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filePointers.ByHash)
	})

	r.HandleFunc("/pending", func(w http.ResponseWriter, r *http.Request) {
		if _, ret := HandleAuth(w, r); ret {
			return
		}

		rawquery := bleve.NewQueryStringQuery("collection:\"No Collection *\"")
		query, err := rawquery.Parse()

		if err != nil {
			RenderError(w, r, err.Error())
			return
		}
		search := bleve.NewSearchRequest(query)
		search.Size = len(filePointers.List)
		search.Fields = []string{"*"}
		searchResults, err := bleveIndex.Search(search)

		if err != nil {
			RenderError(w, r, err.Error())
			return
		}

		ret := map[string]string{}
		for _, thing := range searchResults.Hits {
			value, exists := thing.Fields["title"]
			if exists {
				ret[thing.ID] = value.(string)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ret)
	})

	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		http.ServeFile(w, r, "static/favicon.ico")
	})

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("uwu")
	fmt.Printf("Listening on http://%s:%d\n", config.ServerHost, config.ServerPort)
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", config.ServerHost, config.ServerPort), r)
	if err != nil {
		fmt.Println(err)
		os.Exit(4)
	}
	fmt.Println("owo")
}

func ManageUsers() {
	var username string
	fmt.Print("Username: ")
	fmt.Scanln(&username)
	fmt.Print("Password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Print("**************")
	fmt.Println()

	if err != nil {
		fmt.Println(err)
		os.Exit(4)
	}

	if len(password) < 6 {
		fmt.Println("Password should have at least 6 characters")
		os.Exit(3)
	}

	hashedPassword, err := HashPassword(string(password))

	if err != nil {
		fmt.Println(err)
		os.Exit(7)
	}

	if config.Users == nil {
		config.Users = make(map[string]string)
	}
	_, exists := config.Users[username]

	config.Users[username] = hashedPassword

	newFile, err := config.Export()
	if err != nil {
		fmt.Println(err)
		os.Exit(10)
	}

	fmt.Println("Updated config file:")
	fmt.Println(newFile)

	var overwrite bool
	for {
		var decision string
		fmt.Printf("Overwrite %s with this? [Y/n] ", configPath)
		fmt.Scanln(&decision)

		decision = strings.ToLower(decision)

		if decision == "" || decision == "y" || decision == "yes" {
			overwrite = true
			break
		}

		if decision == "n" || decision == "no" {
			overwrite = false
			break
		}
	}

	if overwrite {
		config.Save(configPath)

		if err != nil {
			fmt.Println(err)
			os.Exit(9)
		}

		if exists {
			fmt.Printf("Password for user %s changed and saved\n", username)
		} else {
			fmt.Printf("User %s created and saved\n", username)
		}
	}
}
