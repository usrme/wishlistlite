# Wishlist Lite

The lesser SSH directory ✨

I made this a little while after discovering the canonical [Wishlist](https://github.com/charmbracelet/wishlist), but after not getting it to work properly (it wasn't able to connect to any of my defined hosts) I still felt the need for a tool of this sort. This luckily coincided with the time I started learning about Go and after a bit of faffing about I got something to work!

## Installation

- using `go install`:

```bash
go install github.com/usrme/wishlist-lite@latest
```

- build it yourself (requires Go 1.18+):

```bash
git clone https://github.com/usrme/wishlist-lite.git
cd wishlist-lite
go build
```

## Usage

At the moment there is only one way to use it and that is to just execute `wishlist-lite`. This opens up an alternate screen where you can (hopefully) see all of your hosts from your `~/.ssh/config` listed. Hosts starting with an asterisk are excluded as those (in my use case) usually mean either a `ProxyJump` or a `User` declaration right after.

Before starting the execution there is a verification that is made that the `ssh` executable exists, but there are still some other assumptions that are made:

- any necessary SSH keys are already loaded into an SSH agent;
- the SSH configuration is similar to this:

```text
Host tst001
  HostName tst001.example.com

Host tst002
  HostName tst002.example.com

Host tst003
  HostName tst003.example.com
```

## Acknowledgments

Couldn't have been possible without the work of people in [Charm](https://github.com/charmbracelet).

## License

[MIT](/LICENSE)
