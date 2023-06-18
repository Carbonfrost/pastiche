package httpclient_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	joehttpclient "github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FilterDownloader", func() {

	Context("when JMESPathFilter", func() {

		It("writes to the output of inner downloader", func() {
			testResponse := &joehttpclient.Response{
				&http.Response{
					Body: io.NopCloser(bytes.NewBufferString(`{"a": "120", "b": "240"}`)),
				},
			}

			var buf bytes.Buffer
			d := httpclient.NewFilterDownloader(
				must(httpclient.NewJMESPathFilter("a")),
				joehttpclient.NewDownloaderTo(&buf),
			)

			writer, _ := d.OpenDownload(context.Background(), testResponse)
			_ = testResponse.CopyTo(writer)
			_ = writer.Close()
			Expect(buf.String()).To(Equal(fmt.Sprintf("%q\n", "120")))
		})
	})

})

func must[T any](t T, err any) T {
	if err != nil {
		panic(err)
	}
	return t
}
