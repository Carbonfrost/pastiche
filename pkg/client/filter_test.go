// Copyright 2023, 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	joehttpclient "github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/client"
	"github.com/Carbonfrost/pastiche/pkg/client/clientfakes"

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
				nil,
			)

			writer, _ := d.OpenDownload(context.Background(), testResponse)
			_ = testResponse.CopyTo(writer)
			_ = writer.Close()
			Expect(buf.String()).To(Equal(fmt.Sprintf("%q", "120")))
		})
	})

	Context("when DigFilter", func() {

		It("writes to the output of inner downloader", func() {
			testResponse := &joehttpclient.Response{
				Response: &http.Response{
					Body: io.NopCloser(bytes.NewBufferString(`{"a": { "b": "240"} }`)),
				},
			}

			var buf bytes.Buffer
			d := client.NewFilterDownloader(
				must(client.NewDigFilter("a.b")),
				joehttpclient.NewDownloaderTo(&buf),
				nil,
			)

			writer, _ := d.OpenDownload(context.Background(), testResponse)
			_ = testResponse.CopyTo(writer)
			_ = writer.Close()
			Expect(buf.String()).To(Equal(fmt.Sprintf("%q", "240")))
		})

		DescribeTable("examples", func(dataJSON, query string, expected any) {
			f := must(client.NewDigFilter(query))

			response := new(clientfakes.FakeResponse)
			response.DataStub = func() (any, error) {
				var data any
				_ = json.Unmarshal([]byte(dataJSON), &data)
				return data, nil
			}

			actual, err := f.Search(response)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		},
			Entry("nominal", `{"a": { "b": 3} }`, "a.b", []byte("3")),
			Entry("into array", `[ 1, 2, 3 ]`, "0", []byte("1")),
			Entry("ignore leading dot", `[ 1, 2, 3 ]`, ".0", []byte("1")),
		)
	})

	Context("when XMLFilter", func() {

		It("writes to the output of inner downloader", func() {
			testResponse := &joehttpclient.Response{
				Response: &http.Response{
					Header: http.Header{
						"Content-Type": []string{"application/xml"},
					},
					Body: io.NopCloser(
						bytes.NewBufferString(`<?xml version="1.0" encoding="utf-8"?>
							<rss version="2.0">
								<channel>
									<link> https://example.com/t </link>
								</channel>
								<item>
									<link> https://example.com/post/1 </link>
								</item>
							</rss>
						`),
					),
				},
			}

			var buf bytes.Buffer
			d := client.NewFilterDownloader(
				must(client.NewXPathFilter("//link")),
				joehttpclient.NewDownloaderTo(&buf),
				nil,
			)

			writer, err := d.OpenDownload(context.Background(), testResponse)
			Expect(err).NotTo(HaveOccurred())

			err = testResponse.CopyTo(writer)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Close()
			Expect(err).NotTo(HaveOccurred())

			Expect(buf.String()).To(Equal(
				"<link> https://example.com/t </link>\n<link> https://example.com/post/1 </link>\n",
			))
		})
	})

})

func must[T any](t T, err any) T {
	if err != nil {
		panic(err)
	}
	return t
}
