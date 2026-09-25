// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package workspace represents the workspace for Pastiche
package workspace

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	joeconfig "github.com/Carbonfrost/joe-cli/extensions/config"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/internal/contextkey"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/Carbonfrost/pastiche/pkg/workspace/logs"
	"sigs.k8s.io/yaml"
)

// Workspace represents the information about the Pastiche
// workspace
type Workspace struct {
	// Action provides the action to provide when the workspace is
	// added to a pipeline.
	cli.Action

	ws *joeconfig.Workspace

	files []*config.File
	model *model.Model

	disableValidation bool
}

// Option sets up an option to configure the workspace
type Option interface {
	cli.Action
	apply(*Workspace)
}

type optionFunc func(*Workspace)

func (f optionFunc) Execute(ctx context.Context) error {
	f.apply(FromContext(ctx))
	return nil
}

func (f optionFunc) apply(ws *Workspace) {
	f(ws)
}

var (
	defaultOptions = []Option{
		withDefaultAction(),
	}
)

const packagePrefixColor = cli.Cyan

// New creates a new workspace
func New(opts ...Option) *Workspace {
	ws := &Workspace{}
	ws.ws = joeconfig.NewWorkspace(
		joeconfig.WithEnvProvider(&envProvider{ws: ws}),
	)

	for _, o := range append(defaultOptions, opts...) {
		o.apply(ws)
	}
	return ws
}

func withDefaultAction() optionFunc {
	return func(w *Workspace) {
		w.Action = cli.Pipeline(
			w.ws.Action,
			FlagsAndArgs(),
			ContextValue(w),
		)
	}
}

// FlagsAndArgs adds flags supporting the workspace. Despite its name, which is conventional,
// this action adds to args.
func FlagsAndArgs() cli.Action {
	return cli.AddFlags([]*cli.Flag{
		{Uses: SetDisableValidation()},
	}...)
}

// FromContext gets the Workspace from the context otherwise panics
func FromContext(ctx context.Context) *Workspace {
	return contextkey.Resolve(ctx, contextkey.Workspace).(*Workspace)
}

// ContextValue provides an action that sets the given value into the context.
// The only supported type is *Workspace.
func ContextValue(v *Workspace) cli.Action {
	return cli.WithContextValue(contextkey.Workspace, v)
}

func (w *Workspace) Pipeline() cli.Action {
	return w.Action
}

// Load will load the workspace, returning the error that occurred
// on load.
func (w *Workspace) Load() (*model.Model, error) {
	if err := w.loadExamples(); err != nil {
		return nil, err
	}

	seen := map[string]bool{}

	for _, root := range w.roots() {
		if err := w.loadRoot(root, seen); err != nil {
			return nil, err
		}
	}

	result := model.New(w.files...)
	if !w.disableValidation {
		return result, model.Validate(result)
	}

	return result, nil
}

// Model obtains the model for the workspace. This method implicitly
// loads the workspace and panics when the workspace cannot be loaded.
// Investigate [Workspace.Load] to load while handling load errors
func (w *Workspace) Model() *model.Model {
	if w.model == nil {
		var err error
		w.model, err = w.Load()
		if err != nil {
			panic(err)
		}
	}
	return w.model
}

func (w *Workspace) loadExamples() error {
	w.files = append(w.files, config.BuiltinFiles()...)
	return nil
}

func (w *Workspace) roots() []string {
	roots := []string{
		os.ExpandEnv("$HOME/.pastiche"),
		".pastiche",
	}

	// PASTICHE_PATH is loaded last so that its items take precedence over
	// the user and workspace directories when names collide.
	for _, path := range filepath.SplitList(os.Getenv("PASTICHE_PATH")) {
		if path == "" {
			continue
		}
		roots = append(roots, path)
	}
	return roots
}

func (w *Workspace) loadRoot(path string, seen map[string]bool) error {
	root, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if seen[root] {
		return nil
	}
	seen[root] = true
	return w.loadFiles(root)
}

func (w *Workspace) loadFiles(root string) error {
	rootFS := os.DirFS(root)
	return fs.WalkDir(rootFS, ".", func(name string, d fs.DirEntry, err error) error {
		if d == nil {
			return nil
		}

		// TODO This should follow rules specified in .ignore files instead
		if d.IsDir() && d.Name() == "logs" {
			return fs.SkipDir
		}
		if strings.HasPrefix(d.Name(), "_") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if err != nil {
			return err
		}

		file, err := config.LoadFile(rootFS, name)
		if err != nil {
			if errors.Is(err, config.ErrUnsupportedFileFormat) {
				return nil
			}

			log.Warnf("%s: %v", filepath.Join(root, name), err)
			return nil
		}

		w.files = append(w.files, file)
		return nil
	})
}

func (w *Workspace) Dir() string {
	return w.ws.Dir()
}

func (w *Workspace) ConfigDir() string {
	return w.ws.ConfigDir()
}

func (w *Workspace) environ() map[string]string {
	return map[string]string{
		"PASTICHE_DIR":        w.Dir(),
		"PASTICHE_LOG_DIR":    w.LogDir(),
		"PASTICHE_CONFIG_DIR": w.ConfigDir(),
		"PASTICHE_PATH":       os.Getenv("PASTICHE_PATH"),
	}
}

type envProvider struct {
	ws *Workspace
}

func (e *envProvider) Environ() iter.Seq2[string, string] {
	m := e.ws.environ()
	return func(yield func(string, string) bool) {
		for _, key := range slices.Sorted(maps.Keys(m)) {
			if !yield(key, m[key]) {
				return
			}
		}
	}
}

// Log retrieves the workspace request log.
func (w *Workspace) Log() *logs.Log {
	return logs.New(w.LogDir())
}

func (w *Workspace) LogDir() string {
	logDir := filepath.Join(w.Dir(), ".pastiche", "logs")
	os.MkdirAll(logDir, 0755)

	return logDir
}

// Describe prints the items in the workspace which match the given parameters,
// either as their configuration or, when a list was requested, as the names
// and kinds of the items
func (w *Workspace) Describe(out cli.Writer, p *DescribeParams) error {
	mo, err := w.Load()
	if err != nil {
		return err
	}

	results, err := mo.Search(p.SearchCriteria()).Results()
	if err != nil {
		return err
	}

	var items describeResults
	for item := range results {
		if err := items.add(item); err != nil {
			return err
		}
	}

	// A spec names the items which are expected to exist, so when the other
	// criteria filter them all out, this is an error rather than empty output
	if items.empty() && p.Spec != nil && len(*p.Spec) > 0 {
		if len(p.Tags) > 0 {
			return fmt.Errorf("not found with tags %v: %q", p.Tags, p.Spec.Path())
		}
		return fmt.Errorf("not found: %q", p.Spec.Path())
	}

	switch {
	case p.Tree:
		return treeItems(out, &items)
	case p.List:
		return listItems(out, &items)
	}
	return displayItems(out, &items)
}

type describeNode struct {
	Name           string
	Children       []describeNode
	InlineChildren []describeNode
}

type describeResults struct {
	Schema    string             `json:"$schema,omitempty"`
	Services  []config.Service   `json:"services,omitempty"`
	VarSets   []config.VarSet    `json:"varSets,omitempty"`
	Mixins    []config.Mixin     `json:"mixins,omitempty"`
	Flows     []config.Flow      `json:"flows,omitempty"`
	Resources []config.Resource  `json:"resources,omitempty"`
	Endpoints []describeEndpoint `json:"endpoints,omitempty"`
}

type describeEndpoint struct {
	Method string `json:"method,omitempty"`
	config.Endpoint
}

const (
	treeBranch     = "├── "
	treeLastBranch = "└── "
	treeIndent     = "│   "
	treeLastIndent = "    "
	bullet         = " • "
)

func (d *describeResults) add(item model.Item) error {
	switch it := item.(type) {
	case *model.Service:
		d.Services = append(d.Services, toConfig[config.Service](it))
	case *model.VarSet:
		d.VarSets = append(d.VarSets, toConfig[config.VarSet](it))
	case *model.Mixin:
		d.Mixins = append(d.Mixins, toConfig[config.Mixin](it))
	case *model.Flow:
		d.Flows = append(d.Flows, toConfig[config.Flow](it))
	case *model.Resource:
		d.Resources = append(d.Resources, *toConfig[*config.Resource](it))
	case *model.Endpoint:
		d.Endpoints = append(d.Endpoints, describeEndpoint{
			Method:   it.Method,
			Endpoint: *toConfig[*config.Endpoint](it),
		})
	default:
		return fmt.Errorf("cannot describe %T", item)
	}
	return nil
}

func (d *describeResults) empty() bool {
	return len(d.Services)+len(d.VarSets)+len(d.Mixins)+len(d.Flows)+len(d.Resources)+len(d.Endpoints) == 0
}

func (d *describeResults) items() []describeItem {
	var res []describeItem
	for _, s := range d.Services {
		res = append(res, describeItem{s.Name, model.ItemKindService})
	}
	for _, v := range d.VarSets {
		res = append(res, describeItem{v.Name, model.ItemKindVarSet})
	}
	for _, f := range d.Flows {
		res = append(res, describeItem{f.Name, model.ItemKindFlow})
	}
	for _, r := range d.Resources {
		res = append(res, describeItem{r.Name, model.ItemKindResource})
	}
	for _, e := range d.Endpoints {
		res = append(res, describeItem{endpointName(e.Name, e.Method), model.ItemKindEndpoint})
	}

	slices.SortFunc(res, func(x, y describeItem) int {
		return cmp.Or(
			cmp.Compare(x.Name, y.Name),
			cmp.Compare(x.Kind, y.Kind),
		)
	})
	return res
}

func (d *describeResults) tree() []describeNode {
	var res []describeNode
	for _, s := range d.Services {
		res = append(res, describeNode{Name: s.Name, Children: resourceNodes(s.Resources)})
	}
	for _, v := range d.VarSets {
		res = append(res, describeNode{Name: v.Name})
	}
	for _, f := range d.Flows {
		res = append(res, describeNode{Name: f.Name})
	}
	res = append(res, resourceNodes(d.Resources)...)
	for _, e := range d.Endpoints {
		res = append(res, describeNode{Name: endpointName(e.Name, e.Method)})
	}

	return sortNodes(res)
}

func resourceNodes(resources []config.Resource) []describeNode {
	var res []describeNode
	for _, r := range resources {
		children := resourceNodes(r.Resources)
		inlineChildren := endpointNodes(r)

		// Take the children of the root node instead
		if r.Name == "" {
			res = append(res, children...)
			res = append(res, inlineChildren...)
			continue
		}
		res = append(res, describeNode{Name: r.Name, Children: sortNodes(children), InlineChildren: sortNodes(inlineChildren)})
	}
	return sortNodes(res)
}

func endpointNodes(r config.Resource) []describeNode {
	endpoints := []struct {
		method   string
		endpoint *config.Endpoint
	}{
		{"GET", r.Get},
		{"PUT", r.Put},
		{"POST", r.Post},
		{"DELETE", r.Delete},
		{"OPTIONS", r.Options},
		{"HEAD", r.Head},
		{"TRACE", r.Trace},
		{"PATCH", r.Patch},
	}

	var res []describeNode
	for _, e := range endpoints {
		if e.endpoint != nil {
			res = append(res, describeNode{Name: endpointName(e.endpoint.Name, e.method)})
		}
	}
	return res
}

func sortNodes(nodes []describeNode) []describeNode {
	slices.SortFunc(nodes, func(x, y describeNode) int {
		return cmp.Compare(x.Name, y.Name)
	})
	return nodes
}

func endpointName(name string, method string) string {
	if name == "" {
		return method
	}
	return fmt.Sprintf("%s (%s)", method, name)
}

// fileSchema identifies the results as a configuration file, which is only
// accurate when every item within it can be written as one
func (d *describeResults) fileSchema() string {
	if len(d.Resources) > 0 || len(d.Endpoints) > 0 {
		return ""
	}
	return config.SchemaFile
}

func toConfig[V any](item model.Item) V {
	return model.ToConfig(item).(V)
}

func displayItems(out io.Writer, d *describeResults) error {
	d.Schema = d.fileSchema()

	data, err := yaml.Marshal(d)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(data))
	return err
}

type describeItem struct {
	Name string
	Kind model.ItemKind
}

// listItems prints the name and kind of each item in two columns
func listItems(out cli.Writer, d *describeResults) error {
	// var previous string
	names := &nameWriter{out: out}

	for _, item := range d.items() {
		if err := names.WriteName(item.Name); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "\t%s\n", item.Kind); err != nil {
			return err
		}
	}
	return nil
}

func treeItems(out cli.Writer, d *describeResults) error {
	names := &nameWriter{out: out}

	for _, node := range d.tree() {
		if err := names.WriteName(node.Name); err != nil {
			return err
		}
		if err := writeInlineNodes(out, node.InlineChildren); err != nil {
			return err
		}
		if _, err := out.WriteString("\n"); err != nil {
			return err
		}
		if err := writeNodes(out, node.Children, ""); err != nil {
			return err
		}
	}
	return nil
}

func writeNodes(out cli.Writer, nodes []describeNode, indent string) error {
	names := &nameWriter{out: out}

	for i, node := range nodes {
		branch, childIndent := treeBranch, treeIndent
		if i == len(nodes)-1 {
			branch, childIndent = treeLastBranch, treeLastIndent
		}

		if _, err := out.WriteString(indent + branch); err != nil {
			return err
		}
		if err := names.WriteName(node.Name); err != nil {
			return err
		}
		if err := writeInlineNodes(out, node.InlineChildren); err != nil {
			return err
		}
		if _, err := out.WriteString("\n"); err != nil {
			return err
		}

		if err := writeNodes(out, node.Children, indent+childIndent); err != nil {
			return err
		}
	}
	return nil
}

func writeInlineNodes(out cli.Writer, nodes []describeNode) error {
	names := &nameWriter{out: out}

	for i, node := range nodes {
		if i == 0 {
			out.WriteString("    ")
		}
		if i > 0 {
			if _, err := out.WriteString(bullet); err != nil {
				return err
			}
		}
		if err := names.WriteName(node.Name); err != nil {
			return err
		}

	}
	return nil
}

// nameWriter writes the names of the items in a sequence of siblings.  Only
// the first name in a run of names which share a package prefix is stylized,
// which makes it apparent where each package begins.
type nameWriter struct {
	out      cli.Writer
	previous string
}

func (w *nameWriter) WriteName(name string) error {
	prefix, rest := cutPackagePrefix(name)
	stylize := prefix != w.previous
	w.previous = prefix

	if err := writePackagePrefix(w.out, prefix, stylize); err != nil {
		return err
	}

	_, err := w.out.WriteString(rest)
	return err
}

func writePackagePrefix(out cli.Writer, prefix string, stylize bool) error {
	if stylize && prefix != "" {
		out.SetForeground(packagePrefixColor)
		defer out.Reset()
	}

	_, err := out.WriteString(prefix)
	return err
}

func cutPackagePrefix(name string) (prefix string, rest string) {
	pkg, after, ok := strings.Cut(name, "/")
	if !ok {
		return "", name
	}
	return pkg + "/", after
}

func DisableValidation() Option {
	return optionFunc(func(w *Workspace) {
		w.disableValidation = true
	})
}

var _ joeconfig.EnvProvider = (*envProvider)(nil)
