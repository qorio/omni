# Embedfs

This is a utility for packaging a directory hierarchy as resources within a single golang executable. 
The utility works by generating golang source code that contains the binary data of the files to be
embedded (e.g. a png image).  The embedded file can be accessed via `import <path>` statements in
your go program as if opening a directory (e.g. `net/http` library's `http.Dir("path/on/disk")`),
except the file content is embedded inside the go executable.  

There's no dependency on another package.  The utility generates code that implements the following
interfaces, in addition to the actual binary data of the embedded files:

+ `os.FileInfo`
+ `http.File`
+ `http.FileSystem`

As an example, Twitter's [Bootstrap](https://github.com/twbs/bootstrap) example site (http://examples.getbootstrap.com/) 
is included and can be built as a single executable. The example, `bootstrap-examples-master.go` accesses the 
embedded files like so:

    import (
	bootstrap_examples "github.com/qorio/embedfs/examples/bootstrap-examples-master"
    )

    // ....
	
    fs := bootstrap_examples.Dir(".")
    http.Handle("/", http.FileServer(fs))


## Building

The embedfs command (`main.go`) itself depends on one of the source files (`pkg/embedfs/fs.go`) to be
packaged within the binary -- so that it can generate the filesystem api implementations.

The embedded filesystem that the program depends on is in the `resources` directory.
To embed the fs.go source code itself in the executable:

    cd pkg
    go run ../main.go -destDir=../resources -match="/fs\\.go$" -generate=true .

This will generate the go files to be compiled.  Then,

    cd .. # back to where main.go is
    go build -o embedfs main.go


## Running the Twitter Bootstrap Example

The Twitter Bootstrap example site is included in the `examples` folder, along with a simple server
`bootstrap-examples-master.go`.  To embed the entire site into a single server, do

    cd examples
    unzip bootstrap-examples-master.zip
    ../embedfs -generate=true bootstrap-examples-master
    go run ../bootstrap-examples-master.go

To prove that the entire site is embedded into a single executable, just build the example

    go build -o bootstrap-examples-master ../bootstrap-examples-master.go

Then run the resulting executable, `bootstrap-examples-master`:

    ./bootstrap-examples-master

and open the browser at [localhost](http://localhost:7777)
