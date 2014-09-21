package godebug

// func Hello() {
//    In()
//    defer Out()
//}

import (
	"container/list"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func init() {
	http.Handle("/debug/goroutine/", http.HandlerFunc(Index))
	http.Handle("/debug/goroutine/stack", http.HandlerFunc(Stack))
	http.Handle("/debug/goroutine/func", http.HandlerFunc(Func))
}

func Index(w http.ResponseWriter, r *http.Request) {
	if err := indexTmpl.Execute(w, nil); err != nil {
		fmt.Fprint(w, err)
	}
}

// /debug/goroutine/stack?id=goid
func Stack(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	goid, _ := strconv.Atoi(r.FormValue("goid"))
	buf := make([]byte, 1<<20)
	GStack(buf, int64(goid))
	w.Write(buf)
}

// /debug/goroutine/func?name=func
func Func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	funcName := r.FormValue("func")
	debug, _ := strconv.Atoi(r.FormValue("debug"))

	if funcName == "" {
		if debug >= 2 {
			fmt.Fprintf(w, "%s", PrintAllStacks())
		} else {
			fmt.Fprintf(w, "%s", PrintAllGoids())
		}
	} else {
		fmt.Fprintf(w, "%s", PrintStacks(funcName))
	}
}

func GetGoId() int64

func GStack([]byte, int64) int

var (
	funcGoids map[string]*list.List
	funcLock  sync.Mutex
)

func init() {
	funcGoids = make(map[string]*list.List)
}

func Add(funcName string, goid int64) {
	funcLock.Lock()
	defer funcLock.Unlock()

	goids, ok := funcGoids[funcName]
	if !ok {
		goids = list.New()
		funcGoids[funcName] = goids
	}
	goids.PushBack(goid)
}

func Delete(funcName string, goid int64) {
	funcLock.Lock()
	defer funcLock.Unlock()

	goids, ok := funcGoids[funcName]
	if !ok {
		return
	}

	for e := goids.Front(); e != nil; e = e.Next() {
		if e.Value == goid {
			goids.Remove(e)
		}
	}
}

func GetFuncGoids(funcName string) []int64 {
	funcLock.Lock()
	defer funcLock.Unlock()

	goids, ok := funcGoids[funcName]
	if !ok {
		return make([]int64, 0)
	}

	ids := make([]int64, 0, goids.Len())

	for e := goids.Front(); e != nil; e = e.Next() {
		ids = append(ids, e.Value.(int64))
	}

	return ids
}

func PrintStacks(funcName string) string {
	funcLock.Lock()
	defer funcLock.Unlock()

	rets := make([]string, 10)

	goids, ok := funcGoids[funcName]
	if !ok {
		return ""
	}

	for e := goids.Front(); e != nil; e = e.Next() {
		buf := make([]byte, 1<<20)
		n := GStack(buf, e.Value.(int64))
		if n < len(buf) {
			buf = buf[:n]
		}
		rets = append(rets, string(buf))
	}

	return strings.Join(rets, "\n")
}

func PrintAllStacks() string {
	funcLock.Lock()
	defer funcLock.Unlock()

	rets := make([]string, 0, 10)

	for funcName, goids := range funcGoids {
		for e := goids.Front(); e != nil; e = e.Next() {
			rets = append(rets, fmt.Sprintf("func [%s]:", funcName))
			buf := make([]byte, 1<<20)
			n := GStack(buf, e.Value.(int64))
			if n < len(buf) {
				buf = buf[:n]
			}
			rets = append(rets, string(buf))
		}
	}

	return strings.Join(rets, "\n")
}

func PrintAllGoids() string {
	funcLock.Lock()
	defer funcLock.Unlock()

	rets := make([]string, 0, 10)

	for funcName, goids := range funcGoids {
		rets = append(rets, fmt.Sprintf("func [%s:%d]:", funcName, goids.Len()))
		for e := goids.Front(); e != nil; e = e.Next() {
			rets = append(rets, fmt.Sprintf(" %d", e.Value.(int64)))
		}
		rets = append(rets, "\n")
	}

	return strings.Join(rets, "")
}

func GetFuncName() string {
	pc, _, _, ok := runtime.Caller(2)
	if ok {
		return runtime.FuncForPC(pc).Name()
	}

	return ""
}

func In() {
	funcName := GetFuncName()
	Add(funcName, GetGoId())
}

func Out() {
	funcName := GetFuncName()
	Delete(funcName, GetGoId())
}

var indexTmpl = template.Must(template.New("index").Parse(`<html>
<head>
<title>/debug/pprof/</title>
</head>
/debug/pprof/<br>
<br>
<body>
profiles:<br>
<table>
<tr><td align=right>1<td><a href="/debug/goroutine/stack?goid=goid">{{.Name}}</a>
<tr><td align=right>2<td><a href="/debug/goroutine/func">{{.Name}}</a>
<tr><td align=right>3<td><a href="/debug/goroutine/func?func=main.main">{{.Name}}</a>
</table>
<br>
<a href="/debug/goroutine/func?debug=2">full func goroutine stack dump</a><br>
</body>
</html>
`))
