package gee

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
)

type H map[string]interface{}

// abortIndex represents a typical value used in abort functions.
const abortIndex int8 = math.MaxInt8 >> 1

type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	Params map[string]string
	// response info
	StatusCode int
	// middleware
	handlers []HandlerFunc
	index    int8
	// engine pointer
	engine *Engine
	// query result
	SqlRes SqlResponse
}

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

func (c *Context) Next() {
	c.index++
	//s := len(c.handlers)
	for ; c.index < int8(len(c.handlers)); c.index++ {
		c.handlers[c.index](c)
		c.index++
	}
}

func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// FormFile returns the first file for the provided form key.
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	f, fh, err := c.Req.FormFile(name)
	if err != nil {
		return nil, err
	}
	f.Close()
	return fh, err
}

// SaveUploadedFile uploads the form file to specific dst
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

func (c *Context) SetHeaders(w http.ResponseWriter, headers map[string]string) {
	header := w.Header()
	for k, v := range headers {
		if header.Get(k) == "" {
			header.Set(k, v)
		}
	}
}

func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("Access-Control-Allow-Origin", "*")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

func (c *Context) File(filePath string) {
	http.ServeFile(c.Writer, c.Req, filePath)
}

// HTML 支持根据模板文件名选择模板进行渲染
func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Fail(500, err.Error())
	}
}

func (c *Context) Fail(code int, err string) {
	c.index = int8(len(c.handlers))
	c.JSON(code, H{"message": err})
}

func (c *Context) writeContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}

// DataFromReader writes the specified reader into the body stream and updates the HTTP code.
func (c *Context) DataFromReader(code int, contentType string, reader io.Reader, extraHeaders map[string]string) {
	c.Status(code)
	c.writeContentType(c.Writer, []string{contentType})
	c.SetHeaders(c.Writer, extraHeaders)
	_, err := io.Copy(c.Writer, reader)
	if err != nil {
		log.Fatal("reader error, ", err)
	}
}

// ShouldBindWith binds the passed struct pointer using the specified binding engine.
// See the binding package.
func (c *Context) ShouldBindWith(obj any, b Binding) error {
	return b.Bind(c.Req, obj)
}

// ShouldBindJSON is a shortcut for c.ShouldBindWith(obj, binding.JSON).
func (c *Context) ShouldBindJSON(obj any) error {
	return c.ShouldBindWith(obj, JSON)
}
