# Changelog

## v0.4.0 (April 1, 2026)

## New features

* Dashboard UI and metadata service
    * Dashboard app - prototype version (43b8c06)
    * Include name on service page (d6433d9)
    * Sort model services (bbeb62f)
    * Rework template loading FS; layout views (a3ea544)
    * Router: skip dirs and files with underscore prefix (a926513)
    * Introduce servers to service page (38de6f8)
    * Introduce sidebar layout; toggle theme (8e660cc)
    * Feature service view; header and var tables; css (9d1c63f)
    * Define Apple touch favicons (7040a24)
    * Introduce nodes to dashboard index view (c9e02e2)
    * Design: services icon (4024c47)
    * Display resources and endpoints; design cleanup (dce6623)
    * Ensure server setup only applies when the command is called (26f9968)
    * Initial server implementation (b4809b9)
* Output filter definitions on resources and as providers
    * Raw filter (7ef142c)
    * Xpath filter (f56711c)
    * Always include metadata on template filter (174b779)
    * Update default filter to pretty print (477c80f)
    * Rework response filtering with new interface; XML support (7af7732)
    * YAML filter (035f547)
    * Introduce gotpl filter result (9b2d58e)
    * Dig filter (cf38c0d)
    * Allow leading dot in dig filter (3a49d1e)
    * Introduce JSON filter (cb219cd)
    * Named output (0acd59e)
    * Allow include metadata on named output filters (aaf7f6f)
    * Filter improvements; include metadata option (8db10b0)
    * Terminal funcs (78a4497)
    * Define base64 funcs for use in template content (78038b2)
    * Bug fix: set default JMESPath query (c4cf296)
* Introduce Request API
    * Expose expander from `Request` (3d95655)
    * Remove `Request` interface (f04319b)
    * Push cascade calculation into `ResolvedResource` API (75f33d5)
    * Split out service resolve from request evaluation (161ef63)
* Update internal catalog of services
    * Add GraphQL introspection to internal third-party services (c0a63a9)
    * Parse built-in configurations (59bfacc)
    * Update the Pastiche metadata service definition (301e3b4)
* Config and model API (b0b066b)
    * Add Name to config.File (365bbf3)
    * Tags (e28bc8c)
    * Auth (6d3f15c)
    * Vars (d2cd263)
    * Omit additional empty JSON attributes (7dbcf85)
    * Add Type field to link (742f326)
    * Provide dual encoding of config.Header (01b150a)
    * Expand using recursive syntax (0e40069)
    * Dynamic client resolution (bb688aa)
    * Split models from binding code (ee7b583)
    * Introduce Title to Resource config (bca056d)
    * Introduce title, description to server config (dc92bc0)
    * Introduce HRefLang attribute to links (42e172d)
    * Allow links to be templates; expand links (b91dd25)
    * Service spec path format (9978fcd)
    * Bug fix: body, rawBody should be untyped in JSON model (4b16648)
    * Convert from model to config (5f391e2)
    * Allow services to contain links (765c821)
    * Reduce vars: implement reduce vars (137683f)
    * Reduce header: implement reduce headers supporting merge and delete (9aefb36)
    * Reduce auth: support Basic merging (d8d086f)
    * Introduce case insensitive matching on method names when converting to config (c79fa36)
    * Change model API to list of services (6abbfe3)
    * Allow configuration to include untyped Body nodes (e873b75)
    * Introduce Lineage; fix to merge nested headers (53d6db2)
    * Consolidate Header calculation in merged resource (6905a77)
    * Expand variables in JSON content (004464b)
    * Allow sourcing configuration files:
        * Allow absolute URLs in fixup relative sources (caa495d)
        * Bug fix: propagate relative file paths when including recursively (356e067)
        * Sourcing partial files (8e0dc78)
        * Allow sourcing in services listed in a config file (230da98)
        * Bug fix: ensure included file marshal errors propagate (1eddeb7)
    * Rename Header to Headers for congruency with config API (c57caea)
    * Expand header variables (0b29f55)
    * Support propagating variables into URI and base URI expansion; set empty string on root (523ced0)
    * Validate models (6d9b1e4)
    * Bug fix: ensure server headers propagate from configuration (3dc3bff)
    * Bug fix: safer handling of unsupported file formats (1600cbb)
    * Improve config file read errors, skip log files (e76125c)
    * Support multiple services in a file (73e5a7e)
    * Add Title attribute to Endpoint (ba2e9bf)
    * Split out LoadFile function; tests (6564fd4)
* Workspace API:
    * Workspace log dir (f7eff5f)
* Richer service definitions:
    * Allow @ to prefix service names; unix URLs (249e88f)
* gRPC support:
    * Initial gRPC client support (c00f6c7)
    * gRPC reflection, TLS support (1367aa9)
    * Reify grpc status errors (c499093)
    * Propagate vars to gRPC request evaluation (7622ebe)
    * Relocate HTTP commands into client package (f08334d)
    * Interop with httpclient headers (bf11e70)
    * Fix relative protoset paths; relocate valid examples (24778e7)
* URL encoded form content (d4a5984)
* History logging download (16eeb07)

## Bug fixes and improvements

* Command line improvements:
    * Revamp describe command (26c6e5b)
    * Allow template variables to the open command (d247f1f)
    * Allow server and method to be used in import command (8998f06)
    * Encapsulate request as binding pattern (2d94517)
    * Workspace clear logs command (8f45b85)
    * Remove persistent HTTP flags to fetch (71a5ceb)
    * Consolidate client fetch and print command (ebc8e93)
    * Add JSON metadata target (dccfd6e)
    * Support JSON marshal dump (7bac568)
    * Open (53fee9a)
    * Display services in YAML (ca45d3c)
    * Import fetch calls:
        * Parse fetch call expressions (3ee8d10)
        * Remove category from fetch command (4253a02)
        * Default to GET method when importing fetch calls (31a5ff0)
        * Improve fetch call parsing (3baf46a)
    * Consolidate persistent HTTP flags error handling (0cbc1a9)
    * Support files in param options (8f9f659)
    * Add CLI option to change client type (a93857a)
    * Dynamically select outbound client type; encapsulate default client command (741b690)
    * Encapsulate init as command pattern (5b14e65)
    * Use a template binding for rendering services help screen (28a42d5)
* Service resolution improvements:
    * Expose base URL from service resolver (b3301d8)
    * Relocate method-context extraction to thunk (9f49d34)
    * Encapsulate client Option pattern; service resolver (c400c83)

* Rename httpclient to just client (559506c)
* Test improvements:
    * Suppress stderr traces in test (40d3f7e)
    * Remove some redundant unit tests (a4d27f0)
* Chores:
    * Update to go1.26 (bda84fa)
    * Update dependent versions (01bbd7f)
    * Update dependent versions (93c84ad)
    * Update to latest joe-cli and joe-cli-http
        * Modernization to remove BindContext; Apply pattern (45825c1)
        * Upgrade dependent versions: joe-cli-http (c450901)
        * Update to include a now passing test (c3c81d9)
        * Use upstream environ expander (a56ae73)
        * Update dependent versions; encapsulate Pastiche-based Location as Middleware (21f3d17)
    * Relocate default user agent string into internal (20ed805)
    * Addresses linter errors; documentation (4005000)
    * go.mod: Update ignore filter (87380f7)
    * Update some documentation comments (9121ff3)
    * Configuration management:
        * Fix broken test build (c987836)
        * Fix github configuration: CI go version (bc366a6)
* Bug fix: no trailing slash in resolved URLs (2de86e5)



## v0.3.0 (November 2, 2025)

## New features

* Allow body to be specified and resolved in configuration (2414c01)
* Allow templates in body content (a820350)
* Allow implicit template var assignments in the fetch command (9e19068)
* Allow URLs to be specified to use `wig`-like behavior (506446a)
* Support completing arguments passed into a service (a0b02a7)
* Support flexibly marshaling Header from YAML (cdb1afa)
* Add links to API and model (4daf42d)
* Filter support; JMESPath filter (7740fa4)
* Introduce `Query` endpoint for use with resources (8ac4db8)
* Support copying headers from `Server` configuration (d4feb13)

## Bug fixes and improvements

* Bug fix: Ensure cumulative paths in resources (5026d8d)
* Bug fix: Address typos in Link tags (54419b0)
* Bug fix: Prevent 'fetch' command being extected within describe (decce50)
* Bug fix: Disallow persistent flags in certain cases; test (1637d87)
* Remove table formatting from help screen (1240dbc)
* Refactoring:
    * Encapsulate model resolution logic in model package (eef9ba0)
    * Consolidate log logic (0873b76)
    * Adopt K8S sigs YAML parsing, simplify tagging (2ab5b79)
    * Unembed Resource from config.Service (6c7c3c1)
* Chores:
    * Tools configuration (43a2e7a)
    * Support reporting Pastiche version via metadata (14ea6a0)
    * Add copyright notices (1618a98)
    * Introduce license file (0bf24ab)
    * Update radstub (5a6ba3d)
    * Adopt code of conduct (92cc249)
    * GitHub configuration:
        * Drop build of version go1.19 (e087be1)
* Dependabot (764ff0e)
    * Update dependent versions:
        * Update engineering platform (a1fba92)
        * Update dependent versions (112de83, 1f61b49, 4ef750f)
        * Bump actions/checkout from 3 to 4 (#6) (1752ed9)
        * Bump goreleaser/goreleaser-action from 4 to 5 (#7) (4179249)
        * Bump actions/setup-go from 3 to 4 (#1) (20f52fc)

## v0.2.0 (March 14, 2023)

### New features

* Allow headers to be specified in server, endpoint, resource (4bd601b)
* Allow looser header JSON marshaling (472c469)
* Support loading Pastiche config from YAML (5813185)
* Detect use of `-X` to change endpoint semantics (0c18e0a)
* Introduce `model` package to encapsulate model semantics (9b4c62a)

### Bug fixes and improvements

* Fetch command; remove HTTP method sub-commands (55673fd)
* Implicitly create GET endpoint in model (9763ccb)
* Port init command to use templates (b6f7648)
* Chores:
    * Update engineering platform (7a1a963)
    * Unit tests for `pasticheMiddleware` (e16cd6d)
    * Upgrade Ginkgo versions (6f03155)
    * Upgrade from YAML v2 to v3 (678771d)
    * Update Go version, `x/net` version, CI (ec5f200)
    * Update dependent versions (66483fc)
    * GitHub CI configuration (67de4b2)

* Service resolver handling of headers (ffef320)
* Bug fix: empty servers list should not panic (55549e6)

## v0.1.0 (July 10, 2022)

* Initial version :sunrise:
