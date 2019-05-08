## Install on Linux

### Install / Update Go

Download latest go binary from [golang.org/dl](https://golang.org/dl/). In this case, version `1.11.5`.

Remove any existing installation, run:

    if [ -d /usr/local/go ]; then sudo rm -r /usr/local/go; fi

Extract to `/usr/local`, run:

    sudo tar -C /usr/local -xzf go1.11.5.linux-amd64.tar.gz

Create go home dir if doesn't already exist, run:

    if [ ! -d $HOME/go ]; then mkdir $HOME/go; fi

Edit `~/.profile`, run:

    nano ~/.profile

Append the following, then save/exit:

    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$(go env GOPATH)
    export PATH=$PATH:$GOPATH/bin

Source updated profile, run:

    source ~/.profile

### Install / Update Dep

To install `dep` for dependency management, run:

    curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

For other OS/environments, refer to the [Dep Installation Documentation](https://golang.github.io/dep/docs/installation.html).
