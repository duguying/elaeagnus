package ela

import (
	"fmt"
	"github.com/codegangsta/inject"
	"github.com/gogather/com/log"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

var (
	middlewares []interface{}
)

func Use(middleware interface{}) {
	t := reflect.TypeOf(middleware)
	if t.Kind() == reflect.Func {
		result, err := injectFuc(middleware)
		if err != nil {
			log.Redf("injection failed: %s\n", err)
		} else {
			middleware = result[0]
		}
	}
	middlewares = append(middlewares, middleware)
}

func injectFuc(fun interface{}, mid ...interface{}) ([]reflect.Value, error) {
	inj := inject.New()
	for i := 0; i < len(middlewares); i++ {
		inj.Map(middlewares[i])
	}
	for i := 0; i < len(mid); i++ {
		inj.Map(mid[i])
	}
	return inj.Invoke(fun)
}

func servHTTP(port int) {
	log.Pinkf("[ela] Listen Port %d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), &elaHandler{})
	if err != nil {
		fmt.Printf("HTTP Server Start Failed Port [%d]\n%s", port, err)
	}
}

// Http handler
type elaHandler struct{}

func (*elaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// package ctx
	ctx := Context{}
	ctx.w = NewResponseWriter(w)
	ctx.r = r
	ctx.Data = make(map[string]interface{})
	ctx.SetStatus(200)

	// ctx as middleware
	Use(ctx)

	// parse router and call action
	path := parseURI(r.URL.String())

	// recording request log
	requestLog(ctx)

	// add special static files
	addSpecialStatic("/favicon.ico")
	addSpecialStatic("/robots.txt")

	// deal with special static files
	for i := 0; i < len(specialStatic); i++ {
		special := specialStatic[i]
		if path == special {
			if staticExist(path) {
				staticServ(path, ctx)
			} else {
				servError(ctx, "<h2>404, File Not Exist</h2>", 404, false)
			}

			// recording response log
			responseLog(ctx)
			return
		}
	}

	staticAlias, errDefault := config.GetString("static", "alias")
	// if static-alias exist, using alias uri mode
	if errDefault == nil {
		if strings.HasPrefix(path, "/"+staticAlias) {
			reg := regexp.MustCompile(`^/` + staticAlias)
			rpath := reg.ReplaceAllString(path, "")
			path = reg.ReplaceAllString(path, "/"+staticDirectory)

			if staticExist(rpath) {
				staticServ(rpath, ctx)
			} else {
				servError(ctx, "<h2>404, File Not Exist</h2>", 404, false)
			}

			// recording response log
			responseLog(ctx)
			return
		}

	}

	servController(path, ctx)

	// recording response log
	responseLog(ctx)
}

func servController(path string, ctx Context) {
	controller := getController(path)
	if controller == nil {
		servError(ctx, "<h2>404, File Not Exist</h2>", 404, false)
		return
	}

	routerElement := controller.(uriMode)
	f := routerElement.fun
	params := routerElement.argsMap

	ctx.setURIParam(params)
	if f != nil {

		functions := f

		defer func() {
			if r := recover(); r != nil {

				var stack string
				for i := 1; ; i++ {
					_, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					}
					stack = stack + fmt.Sprintln(fmt.Sprintf("%s:%d", file, line))
				}

				content := "<h2>500 Server Internal Error!</h2>\n\n" + fmt.Sprintf("%s", r) + "\n\n" + "<pre>" + stack + "</pre>"
				log.Redln(r)
				log.Yellowln(stack)

				servError(ctx, content, 500, false)
				return
			}
		}()

		// execute before controllers
		if beforeController != nil && routerElement.withBefore {
			injectFuc(beforeController)
		}

		// execute controllers
		for i := 0; i < len(functions); i++ {
			if !ctx.GetResponseWriter().HasFlushed() {
				function := functions[i]
				injectFuc(function)
			}
		}

		// execute after controllers
		if afterController != nil && routerElement.withBefore {
			injectFuc(afterController)
		}

	} else {
		// if static-alias does not exist, using default mode
		if staticExist(path) {
			staticServ(path, ctx)
		} else {
			servError(ctx, "<h2>404, File Not Exist</h2>", 404, false)
		}
	}
}
