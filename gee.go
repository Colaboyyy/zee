package gee

import (
	"fmt"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"html/template"
	"net/http"
	"os"
	"path"
	"strings"
)

type HandlerFunc func(c *Context)

type RouterGroup struct {
	Handlers    []HandlerFunc
	prefix      string
	middlewares []HandlerFunc
	router      *router
	engine      *Engine
}

type Engine struct {
	*RouterGroup
	groups        []*RouterGroup     // store all groups
	htmlTemplates *template.Template // for html render -- 将所有的模板加载进内存
	funcMap       template.FuncMap   // for html render -- 所有的自定义模板渲染函数
	noRoute       []HandlerFunc
	noMethod      []HandlerFunc
	allNoRoute    []HandlerFunc
	allNoMethod   []HandlerFunc
	// UseH2C enable h2c support.
	UseH2C bool
}

// New is the constructor of gee.Engine
func New() *Engine {
	engine := &Engine{}
	engine.RouterGroup = &RouterGroup{router: newRouter()}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	engine.RouterGroup.engine = engine
	return engine
}

// Group is defined to create a new RouterGroup
// remember all groups share the same Engine instance
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		router: group.router,
	}
	return newGroup
}

func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	//pattern := group.prefix + comp
	pattern := group.calculateAbsolutePath(comp)
	group.router.addRoute(method, pattern, handler)
}

// NoRoute adds handlers for NoRoute. It returns a 404 code by default.
func (engine *Engine) NoRoute(handlers ...HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

func (engine *Engine) NoMethod(handlers ...HandlerFunc) {
	engine.noMethod = handlers
	engine.rebuild405Handlers()
}

func (engine *Engine) rebuild404Handlers() {
	engine.allNoRoute = engine.combineHandlers(engine.noRoute)
}

func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

func (group *RouterGroup) HEAD(pattern string, handler HandlerFunc) {
	group.addRoute("HEAD", pattern, handler)
}

func (engine *Engine) Handler() http.Handler {
	if !engine.UseH2C {
		return engine
	}

	h2s := &http2.Server{}
	return h2c.NewHandler(engine, h2s)
}

func (engine *Engine) RUN(addr string) (err error) {
	return http.ListenAndServe(addr, engine.Handler())
}

func (engine *Engine) Use(middleware ...HandlerFunc) {
	engine.RouterGroup.Use(middleware...)
	engine.rebuild404Handlers()
	engine.rebuild405Handlers()
}

// Use is defined to add middleware to the group
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

func (engine *Engine) rebuild405Handlers() {
	engine.allNoMethod = engine.combineHandlers(engine.noMethod)
}

func (engine *Engine) handleMidderwares(req *http.Request) []HandlerFunc {
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	engine.middlewares = middlewares
	return middlewares
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	var handlers = engine.handleMidderwares(req)
	//c.handlers = append(handlers, engine.allNoRoute...)
	//c.handlers = engine.allNoRoute
	c.handlers = handlers
	c.engine = engine

	engine.router.handle(c, engine)
}

func (group *RouterGroup) combineHandlers(handlers []HandlerFunc) []HandlerFunc {
	finalSize := len(group.Handlers) + len(handlers)
	assert1(finalSize < int(abortIndex), "too many handlers")
	mergedHandlers := make([]HandlerFunc, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}

func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.prefix, relativePath)
}

// create static handler
func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	//absolutePath := path.Join(group.prefix, relativePath)
	absolutePath := group.calculateAbsolutePath(relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(ctx *Context) {

		file := ctx.Param("filepath")
		// Check if file exists and/or if we have permission to access it
		f, err := fs.Open(file)
		if err != nil {
			ctx.Status(http.StatusNotFound)
			ctx.handlers = group.engine.noRoute
			ctx.index = -1
			return
		}
		f.Close()
		fileServer.ServeHTTP(ctx.Writer, ctx.Req)
	}
}

// serve static files
func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	// Register GET handlers
	group.GET(urlPattern, handler)
	group.HEAD(urlPattern, handler)
}

func (group *RouterGroup) StaticFs(relativePath string, fs http.FileSystem) {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}
	handler := group.createStaticHandler(relativePath, fs)
	urlPattern := path.Join(relativePath, "/*filepath")

	// Register GET and HEAD handlers
	group.GET(urlPattern, handler)
	group.HEAD(urlPattern, handler)
}

// ---HTML---
// SetFuncMap 自定义渲染函数
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

// 加载模板的方法
func (engine *Engine) LoadHTMLGlob(pattern string) {
	fmt.Println(os.Executable())
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

// 默认实例使用 Logger 和 Recovery 中间件。
func Default() *Engine {
	engine := New()
	engine.Use(Logger(), Recovery())
	return engine
}
