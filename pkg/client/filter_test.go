// Copyright 2023, 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package client_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	joehttpclient "github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FilterDownloader", func() {

	Context("when JMESPathFilter", func() {

		It("writes to the output of inner downloader", func() {
			testResponse := &joehttpclient.Response{
				Response: &http.Response{
					Body: io.NopCloser(bytes.NewBufferString(`{"a": "120", "b": "240"}`)),
				},
			}

			var buf bytes.Buffer
			d := client.NewFilterDownloader(
				must(client.NewJMESPathFilter("a")),
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
