# Heptio Workgroup

**Maintainers:** [Heptio][0]

[![Build Status][1]][2]

[![godoc][3]][4]

## Overview
Workgroup is a small utility to manage the lifetime of a set of related goroutines.

## Examples

### http.Serve

```
var g workgroup.Group

g.Add(func(stop <-chan struct{}) error {
	l, err := net.Listen("tcp", ":80") // listen on port 80
        if err != nil {
                return err
        }

        go func() {
                <-stop // close listener on stop request
                l.Close()
        }()
        return http.Serve(l, mux)
})
g.Run()
```

## Related work

`workgroup.Group` is heavily inspired by prior art including oklog's [`run.Group`][9] and Gustavo Niemeyer's [`tomb`][10] packages.

## Contributing

Thanks for taking the time to join our community and start contributing!

Bug reports are most welcome, but with the exception of #5, this project is closed.

* Please familiarize yourself with the [Code of Conduct][8] before contributing.
* See [CONTRIBUTING.md][5] for information about setting up your environment, the workflow that we expect, and instructions on the developer certificate of origin that we require.

## Changelog

See [the list of releases][6] to find out about feature changes.

[0]: https://github.com/heptio
[1]: https://travis-ci.org/heptio/workgroup.svg?branch=master
[2]: https://travis-ci.org/heptio/workgroup
[3]: https://godoc.org/github.com/heptio/workgroup?status.svg
[4]: https://godoc.org/github.com/heptio/workgroup
[5]: /CONTRIBUTING.md
[6]: https://github.com/heptio/contour/releases
[8]: /CODE_OF_CONDUCT.md
[9]: https://github.com/oklog/run
[10]: https://godoc.org/gopkg.in/tomb.v2
