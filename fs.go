package GoMatrix

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 解决304状态码，改造了http/fs源码

var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&#34;",
	"'", "&#39;",
)
var errNoOverlap = errors.New("invalid range: failed to overlap")


type (
	Dir string
	condResult int
	dirEntryDirs []fs.DirEntry
	fileInfoDirs []fs.FileInfo
	countingWriter int64
)

const (
	condNone condResult = iota
	condTrue
	condFalse
)

const sniffLen = 512

type anyDirs interface {
	len() int
	name(i int) string
	isDir(i int) bool
}

type httpRange struct {
	start, length int64
}

func (d dirEntryDirs) len() int          { return len(d) }
func (d dirEntryDirs) isDir(i int) bool  { return d[i].IsDir() }
func (d dirEntryDirs) name(i int) string { return d[i].Name() }

func (d fileInfoDirs) len() int          { return len(d) }
func (d fileInfoDirs) isDir(i int) bool  { return d[i].IsDir() }
func (d fileInfoDirs) name(i int) string { return d[i].Name() }


func (d Dir) Open(name string) (http.File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("http: invalid character in file path")
	}
	dir := string(d)
	if dir == "" {
		dir = "."
	}
	fullName := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))
	f, err := os.Open(fullName)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (c *Context)ServeFile(name string)  {
	if containsDotDot(c.Req.URL.Path) {
		http.Error(c.Writer, "invalid URL path", http.StatusBadRequest)
		return
	}
	dir, file := filepath.Split(name)
	c.serveFile(Dir(dir), file, false)
}

func (c *Context)serveFile(fs http.FileSystem, name string, redirect bool) {
	const indexPage = "/index.html"
	if strings.HasSuffix(c.Req.URL.Path, indexPage) {
		c.localRedirect("./")
		return
	}
	f, err := fs.Open(name)
	if err != nil {
		msg, code := toHTTPError(err)
		http.Error(c.Writer,msg,code)
		return
	}
	defer f.Close()
	d, err := f.Stat()
	if err != nil {
		msg, code := toHTTPError(err)
		http.Error(c.Writer, msg, code)
		return
	}

	if redirect {
		url := c.Req.URL.Path
		if d.IsDir() {
			if url[len(url)-1] != '/' {
				c.localRedirect(path.Base(url)+"/")
				return
			}
		} else {
			if url[len(url)-1] == '/' {
				c.localRedirect("../"+path.Base(url))
				return
			}
		}
	}
	if d.IsDir() {
		url := c.Req.URL.Path
		if url == "" || url[len(url)-1] != '/' {
			c.localRedirect(path.Base(url)+"/")
			return
		}

		index := strings.TrimSuffix(name, "/") + indexPage
		ff, err := fs.Open(index)
		if err == nil {
			defer ff.Close()
			dd, err := ff.Stat()
			if err == nil {
				name = index
				d = dd
				f = ff
			}
		}
	}

	if d.IsDir() {
		if c.checkIfModifiedSince(d.ModTime()) == condFalse {
			c.writeNotModified()
			return
		}
		c.setLastModified(d.ModTime())
		c.dirList(f)
		return
	}
	sizeFunc := func() (int64, error) { return d.Size(), nil }
	c.serveContent(sizeFunc, f, d.Name(), d.ModTime())
}

func (c *Context)localRedirect(newPath string) {
	if q := c.Req.URL.RawQuery; q != "" {
		newPath += "?" + q
	}
	c.SetHeader("Location", newPath)
	c.Status(http.StatusMovedPermanently)
}

func (c *Context)serveContent(sizeFunc func() (int64, error), content io.ReadSeeker, name string, modtime time.Time,) {
	c.setLastModified(modtime)
	done, rangeReq := c.checkPreconditions(modtime)
	if done {
		return
	}
	code := http.StatusOK
	ctypes, haveType := c.Writer.Header()["Content-Type"]
	var ctype string
	if !haveType {
		ctype = mime.TypeByExtension(filepath.Ext(name))
		if ctype == "" {
			// read a chunk to decide between utf-8 text and binary
			var buf [sniffLen]byte
			n, _ := io.ReadFull(content, buf[:])
			ctype = http.DetectContentType(buf[:n])
			_, err := content.Seek(0, io.SeekStart) // rewind to output whole file
			if err != nil {
				http.Error(c.Writer, "seeker can't seek", http.StatusInternalServerError)
				return
			}
		}
		c.SetHeader("Content-Type", ctype)
	} else if len(ctypes) > 0 {
		ctype = ctypes[0]
	}
	size, err := sizeFunc()
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
	sendSize := size
	var sendContent io.Reader = content
	if size >= 0 {
		ranges, err := parseRange(rangeReq, size)
		if err != nil {
			if err == errNoOverlap {
				c.SetHeader("Content-Range", fmt.Sprintf("bytes */%d", size))
			}
			http.Error(c.Writer, err.Error(), http.StatusRequestedRangeNotSatisfiable)
			return
		}
		if sumRangesSize(ranges) > size {
			ranges = nil
		}
		switch {
		case len(ranges) == 1:
			ra := ranges[0]
			if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
				http.Error(c.Writer, err.Error(), http.StatusRequestedRangeNotSatisfiable)
				return
			}
			sendSize = ra.length
			code = http.StatusPartialContent
			c.SetHeader("Content-Range", ra.contentRange(size))
		case len(ranges) > 1:
			sendSize = rangesMIMESize(ranges, ctype, size)
			code = http.StatusPartialContent
			pr, pw := io.Pipe()
			mw := multipart.NewWriter(pw)
			c.SetHeader("Content-Type", "multipart/byteranges; boundary="+mw.Boundary())
			sendContent = pr
			defer pr.Close() // cause writing goroutine to fail and exit if CopyN doesn't finish.
			go func() {
				for _, ra := range ranges {
					part, err := mw.CreatePart(ra.mimeHeader(ctype, size))
					if err != nil {
						pw.CloseWithError(err)
						return
					}
					if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
						pw.CloseWithError(err)
						return
					}
					if _, err := io.CopyN(part, content, ra.length); err != nil {
						pw.CloseWithError(err)
						return
					}
				}
				mw.Close()
				pw.Close()
			}()
		}
		c.SetHeader("Accept-Ranges", "bytes")
		if c.GetHeader("Content-Encoding") == "" {
			c.SetHeader("Content-Length", strconv.FormatInt(sendSize, 10))
		}
	}
	c.Status(code)
	if c.Req.Method != "HEAD" {
		io.CopyN(c.Writer, sendContent, sendSize)
	}
}

func parseRange(s string, size int64) ([]httpRange, error) {
	if s == "" {
		return nil, nil
	}
	const b = "bytes="
	if !strings.HasPrefix(s, b) {
		return nil, fmt.Errorf("invalid range")
	}
	var ranges []httpRange
	noOverlap := false
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = textproto.TrimString(ra)
		if ra == "" {
			continue
		}
		i := strings.Index(ra, "-")
		if i < 0 {
			return nil, fmt.Errorf("invalid range")
		}
		start, end := textproto.TrimString(ra[:i]), textproto.TrimString(ra[i+1:])
		var r httpRange
		if start == "" {
			if end == "" || end[0] == '-' {
				return nil, fmt.Errorf("invalid range")
			}
			i, err := strconv.ParseInt(end, 10, 64)
			if i < 0 || err != nil {
				return nil, fmt.Errorf("invalid range")
			}
			if i > size {
				i = size
			}
			r.start = size - i
			r.length = size - r.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return nil, fmt.Errorf("invalid range")
			}
			if i >= size {
				noOverlap = true
				continue
			}
			r.start = i
			if end == "" {
				r.length = size - r.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.start > i {
					return nil, fmt.Errorf("invalid range")
				}
				if i >= size {
					i = size - 1
				}
				r.length = i - r.start + 1
			}
		}
		ranges = append(ranges, r)
	}
	if noOverlap && len(ranges) == 0 {
		return nil, fmt.Errorf("invalid range: failed to overlap")
	}
	return ranges, nil
}

func (c *Context)setLastModified(modtime time.Time) {
	if !isZeroTime(modtime) {
		c.SetHeader("Last-Modified", modtime.UTC().Format(TimeFormat))
	}
}

func isZeroTime(t time.Time) bool {
	var unixEpochTime = time.Unix(0, 0)
	return t.IsZero() || t.Equal(unixEpochTime)
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

func containsDotDot(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func (c *Context)checkIfModifiedSince(modtime time.Time) condResult {
	if c.Req.Method != "GET" && c.Req.Method != "HEAD" {
		return condNone
	}
	ims := c.GetHeader("If-Modified-Since")
	if ims == "" || isZeroTime(modtime) {
		return condNone
	}
	t, err := http.ParseTime(ims)
	if err != nil {
		return condNone
	}
	modtime = modtime.Truncate(time.Second)
	if modtime.Before(t) || modtime.Equal(t) {
		return condFalse
	}
	return condTrue
}

func (c *Context)writeNotModified() {
	h := c.Writer.Header()
	delete(h, "Content-Type")
	delete(h, "Content-Length")
	if h.Get("Etag") != "" {
		delete(h, "Last-Modified")
	}
	c.Status(http.StatusOK)
}

func logf(r *http.Request, format string, args ...interface{}) {
	s, _ := r.Context().Value(http.ServerContextKey).(*http.Server)
	if s != nil && s.ErrorLog != nil {
		s.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func (c *Context)dirList(f http.File) {
	var dirs anyDirs
	var err error
	if d, ok := f.(fs.ReadDirFile); ok {
		var list dirEntryDirs
		list, err = d.ReadDir(-1)
		dirs = list
	} else {
		var list fileInfoDirs
		list, err = f.Readdir(-1)
		dirs = list
	}

	if err != nil {
		logf(c.Req, "http: error reading directory: %v", err)
		http.Error(c.Writer, "Error reading directory", http.StatusInternalServerError)
		return
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs.name(i) < dirs.name(j) })

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(c.Writer, "<pre>\n")
	for i, n := 0, dirs.len(); i < n; i++ {
		name := dirs.name(i)
		if dirs.isDir(i) {
			name += "/"
		}
		url := url.URL{Path: name}
		fmt.Fprintf(c.Writer, "<a href=\"%s\">%s</a>\n", url.String(), htmlReplacer.Replace(name))
	}
	fmt.Fprintf(c.Writer, "</pre>\n")
}

func (c *Context)checkPreconditions(modtime time.Time) (done bool, rangeHeader string) {
	rangeHeader = c.GetHeader("Range")
	if rangeHeader != "" && c.checkIfRange(modtime) == condFalse {
		rangeHeader = ""
	}
	return false, rangeHeader
}

func scanETag(s string) (etag string, remain string) {
	s = textproto.TrimString(s)
	start := 0
	if strings.HasPrefix(s, "W/") {
		start = 2
	}
	if len(s[start:]) < 2 || s[start] != '"' {
		return "", ""
	}
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		switch {
		case c == 0x21 || c >= 0x23 && c <= 0x7E || c >= 0x80:
		case c == '"':
			return s[:i+1], s[i+1:]
		default:
			return "", ""
		}
	}
	return "", ""
}

func etagStrongMatch(a, b string) bool {
	return a == b && a != "" && a[0] == '"'
}

func (c *Context)checkIfRange(modtime time.Time) condResult {
	if c.Req.Method != "GET" && c.Req.Method != "HEAD" {
		return condNone
	}
	ir := c.GetHeader("If-Range")
	if ir == "" {
		return condNone
	}
	etag, _ := scanETag(ir)
	if etag != "" {
		if etagStrongMatch(etag, c.GetHeader("Etag")) {
			return condTrue
		} else {
			return condFalse
		}
	}
	if modtime.IsZero() {
		return condFalse
	}
	t, err := http.ParseTime(ir)
	if err != nil {
		return condFalse
	}
	if t.Unix() == modtime.Unix() {
		return condTrue
	}
	return condFalse
}

func sumRangesSize(ranges []httpRange) (size int64) {
	for _, ra := range ranges {
		size += ra.length
	}
	return
}

func (r httpRange) contentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.start, r.start+r.length-1, size)
}

func rangesMIMESize(ranges []httpRange, contentType string, contentSize int64) (encSize int64) {
	var w countingWriter
	mw := multipart.NewWriter(&w)
	for _, ra := range ranges {
		mw.CreatePart(ra.mimeHeader(contentType, contentSize))
		encSize += ra.length
	}
	mw.Close()
	encSize += int64(w)
	return
}

func (w *countingWriter) Write(p []byte) (n int, err error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

func (r httpRange) mimeHeader(contentType string, size int64) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Range": {r.contentRange(size)},
		"Content-Type":  {contentType},
	}
}

func toHTTPError(err error) (msg string, httpStatus int) {
	if errors.Is(err, fs.ErrNotExist) {
		return "404 page not found", http.StatusNotFound
	}
	if errors.Is(err, fs.ErrPermission) {
		return "403 Forbidden", http.StatusForbidden
	}
	// Default:
	return "500 Internal Server Error", http.StatusInternalServerError
}
