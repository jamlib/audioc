## Install on Linux

### Install Go

Download latest go binary from [golang.org/dl](https://golang.org/dl/). In this case, version 1.10.

Extract to `/usr/local`, run:

    sudo tar -C /usr/local -xzf go1.10.linux-amd64.tar.gz

Create go home dir if doesn't already exist, run:

    if [ ! -d $HOME/go ]; then mkdir $HOME/go; fi

Open ~/.profile for editing, run:

    nano ~/.profile

Append the following, then save/exit:

    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin

Source updated profile, run:

    source ~/.profile

### Install Dep

To install `dep` for dependency management, run:

    curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

For other OS/environments, refer to the [Dep Installation Documentation](https://golang.github.io/dep/docs/installation.html).
