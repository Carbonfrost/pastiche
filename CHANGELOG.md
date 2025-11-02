# Changelog

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
