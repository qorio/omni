package shorty

import (
	"bytes"
	"fmt"
	"testing"
)

func TestGenerateInterstitial(t *testing.T) {

	destination := "url1"
	appstore := "url2"

	rule := &RoutingRule{
		Destination:                     destination,
		AppStoreUrl:                     appstore,
		InterstitialToAppStoreOnTimeout: OnOff("on"),
	}

	context := appInstallInterstitialContext{
		Rule: rule,
		IsCrossBrowserContext: false,
	}

	var buf1, buf2 bytes.Buffer
	deeplinkJsTemplate.Execute(&buf1, context)
	fmt.Println(buf1.String())

	context.IsCrossBrowserContext = true
	buf1.Reset()

	deeplinkJsTemplate.Execute(&buf1, context)
	fmt.Println(buf1.String())

	context.IsCrossBrowserContext = false

	openTestHtmlTemplate.Execute(&buf2, context)
	fmt.Println(buf2.String())

	context.IsCrossBrowserContext = true
	buf2.Reset()

	openTestHtmlTemplate.Execute(&buf2, context)
	fmt.Println(buf2.String())

}
