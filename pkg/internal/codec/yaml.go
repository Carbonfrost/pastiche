package codec

import (
	"encoding/json"
	"io"

	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
	"sigs.k8s.io/yaml"
)

func init() {
	// Pastiche uses sigs YAML, which routes via JSON for output
	marshal.RegisterCodec(marshal.YAML, func() codec.Interface {
		return &yamlCodec{}
	})
}

type yamlCodec struct {
	opts []yaml.JSONOpt
}

func (y *yamlCodec) MarshalWrite(w io.Writer, in any) error {
	data, err := yaml.Marshal(in)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// UnmarshalRead implements [codec.Interface].
func (y *yamlCodec) UnmarshalRead(r io.Reader, out any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out, y.opts...)
}

func (y *yamlCodec) DisallowUnknownFields() {
	y.opts = append(y.opts, func(d *json.Decoder) *json.Decoder {
		d.DisallowUnknownFields()
		return d
	})
}
