# go-oss

go-oss is a static assets library for [veeka-oss-mirror](https://github.com/olachat/veeka-oss-mirror) repository. It contains constant url for assets in that repository, and is self-updating via Github Actions when the repository updates.

## Installation

To get the latest version use:

```bash
go get github.com/olachat/go-oss@latest
```

If you have added assets, you can simply update the library in your project via go get as above.

If you encounter errors due to unable to download repository (go-oss is a private repository), can 

## Usage

```go
package main

import (
	"fmt"
	oss "github.com/olachat/go-oss/assets/static/veeka/game/bomb"
)

func main() {
	fmt.Println("mosaic url is:", oss.Mosaic)
}
```

## Common Problems

### 1) Unable to run go get on private repository.

You might need to add private repository into your go get paths. Can try the following:

- Check that you have `github.com/olachat` in your go environment variables (GOPRIVATE,GONOPROXY,GONOSUMDB).
  1) run `go env` and verify that the variables contains `github.com/olachat`
  2) if not, add them by running `go env -w GOPRIVATE=github.com/olachat`, then check again
- If you are using ssh authentication, check your .gitconfig reroutes github.com to use ssh
  1) open `~/.gitconfig` and verify that you have:
     ```bash
     [url "git@github.com:"]
         insteadOf = https://github.com/
     ```
  2) if not, add them using vim or nano

### 2) I cannot find my asset after adding library

Check the following:

- After veeka-oss-mirror repository is updated on main branch, it can take awhile before library is updated. To check status, you can check that the actions have recently ran: [go-oss/actions](https://github.com/olachat/go-oss/actions)
- Your import might have conflicting path, as library maps the asset into respective directory. Double check your import path to point to your assets directory.
- You might require different package name if you are importing assets from different directories. 
  - For example, if you require `static/veeka/idRobotIcon` and `static/veeka/thRobotIcon`
  - you need to import them separately and give a different package name:
    ```go
    import (
        idRobotIcon "github.com/olachat/go-oss/assets/static/veeka/idRobotIcon"
        thRobotIcon "github.com/olachat/go-oss/assets/static/veeka/thRobotIcon"
    )
    ```
- If you have named your assets starting with non-alphabet characters, the library might add a `K` prefix
