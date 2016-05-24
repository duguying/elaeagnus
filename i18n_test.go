package ela

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestI18n(t *testing.T) {
	Convey("I18n sections", t, func() {
		i18n := NewI18n("etc/locale")
		i18n.SetLang("zh_CN")

		hello := i18n.Tr("hello")
		world := i18n.Tr("world")

		fmt.Println(hello)
		fmt.Println(world)

		So(hello, ShouldEqual, "你好")
		So(world, ShouldEqual, "世界")

		i18ne := NewEmptyI18n()

		So(i18ne, ShouldNotBeNil)

		i18ne.Load("etc/locale")

		hello1 := i18n.Tr("_", "hello")
		world1 := i18n.Tr("_", "world")

		fmt.Println(hello1)
		fmt.Println(world1)

		So(hello1, ShouldEqual, "你好")
		So(world1, ShouldEqual, "世界")

		mux := http.NewServeMux()
		mux.HandleFunc("/sendstrailers", func(w http.ResponseWriter, req *http.Request) {
			ctx := Context{}
			ctx.w = NewResponseWriter(w)
			ctx.r = req
			ctx.Data = make(map[string]interface{})

			fmt.Println("...")
			Use(&ctx)
			Use(InitI18nModule("etc/locale"))
		})

	})

}
