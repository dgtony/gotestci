# Testing Go projects with CI
Utility for automatic testing of multi-package Go projects in CI pipelines.

## Install

```
go get github.com/dgtony/gotestci
```


## Usage

Run utility from the top directory of the project:

```
cd $GOPATH/src/<path_to_project>
gotestci
```

Utility inspects project structure and tests every subpackage one by one. Typical report has three fields and looks as follows:

```
status: passed, coverage: 79.2%, packages with tests: 25.0%
```

* **status** - overall testing status; if some test failed or any error occured then testing status will be `failed`, otherwise - `passed`
* **coverage** - accurate total coverage for all packages supplied with tests
* **packages with tests** - percent of packages in the project, supplied with tests.

#### Exclude packages
To exclude one or several packages from testing just use `-e` flag, i.e.:

```
gotestci -e github.com/pkg1 -e github.com/pkg2
```

#### Testing progress
In order to see testing progress in percents during the process one can use flag `-p` (disabled by default).

#### Coverage mode
By default utility choose `set` coverage mode. However one can specify other modes, such as `count` and `atomic` with flag `-m`:

```
gotestci -m atomic
```

More information about coverage testing could be found in [The Go blog](https://blog.golang.org/cover).