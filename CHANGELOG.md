# Changelog

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
