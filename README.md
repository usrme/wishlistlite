# Wishlist Lite

[![test](https://github.com/usrme/wishlistlite/actions/workflows/test.yml/badge.svg)](https://github.com/usrme/wishlistlite/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/usrme/wishlistlite)](https://goreportcard.com/report/github.com/usrme/wishlistlite)

The lesser SSH directory ✨

I made this a little while after discovering the canonical [Wishlist](https://github.com/charmbracelet/wishlist), but after not getting it to work properly (it wasn't able to connect to any of my defined hosts) I still felt the need for a tool of this sort. This luckily coincided with the time I started learning about Go and after a bit of faffing about I got something to work!

## Installation

- using `go install`:

```bash
go install github.com/usrme/wishlistlite@latest
```

- download a binary from the [releases](https://github.com/usrme/wishlistlite/releases) page

- build it yourself (requires Go 1.18+):

```bash
git clone https://github.com/usrme/wishlistlite.git
cd wishlistlite
go build
```

## Removal

```bash
rm -f "${GOPATH}/bin/wishlistlite"
rm -rf "${GOPATH}/pkg/mod/github.com/usrme/wishlistlite*"
```

## Usage

Execute `wishlistlite`, which opens up an alternate screen where you can (hopefully) see all of your hosts from your `~/.ssh/config` listed, which can then be SSH-ed into by selecting the appropriate one using arrow keys, or filtering for it, and pressing Enter. It then just grabs whatever is next to the `Host` declaration, precedes it with `ssh`, runs that executable, and exits `wishlistlite` leaving you with an SSH session.

It's also possible to press the letter `i`, which will allow you to supply a host to connect to on an ad-hoc basis. For example inputting `user@example.com` and pressing Enter will go through the exact process as above. Any of the entries from your SSH configuration are still valid as inputs.

To sort by recently connected to hosts press the letter `r`, which will read the file `~/.ssh/recent.json`, which gets created with the same entries as presented in the default view. After connecting to various hosts that file will be sorted according to which host was connected to recently. Press `r` again to return to the default view.

To delete entries from the recently connected view press the letter `d`, which will remove the selected host from the view and immediately save the changes to the `~/.ssh/recent.json` file.

### Caveats

Hosts starting with an asterisk are excluded as those (in my use case) usually mean either a `ProxyJump` or a `User` declaration right after. The entire parsing is done with regular expressions, so there may be other edge cases with parsing, but I've tried to cover the most common cases with the included tests.

Before starting the execution there is a verification that is made that the `ssh` executable exists and that any necessary SSH keys are already loaded into an SSH agent.

The ability to show a stopwatch counting up to the moment a connection is made is achieved through the use of the ['ControlMaster'](https://www.mankier.com/5/ssh_config#ControlMaster), ['ControlPersist'](https://www.mankier.com/5/ssh_config#ControlPersist), and ['ControlPath'](https://www.mankier.com/5/ssh_config#ControlPath) SSH options. Here is some [more information on those options](https://usrme.xyz/tils/that-ssh-allows-for-connection-sharing/) and the [GitHub issue](https://github.com/usrme/wishlistlite/issues/8) behind implementing it. The caveat is in that a socket is being set up in `/dev/shm` that is privileged only to your own user.

## Acknowledgments

Couldn't have been possible without the work of people in [Charm](https://github.com/charmbracelet).

## License

[MIT](/LICENSE)
