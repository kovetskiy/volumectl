# volumectl

The volume control command line interface.

# Why

Because I was tired of other volume control clients that
1) don't do correctly what supposed to do
2) very slow especially when watching videos on Youtube

# What

Two parts: daemon and client, daemon keeps connect with pulseaudio server using
Unix socket. Client talks to daemon what to do and ask for some information.

Every time when you change volume, other implementations re-request server
information, current volume on every command, this implementation doesn't do
this, it obtains information one time after connect to pulseaudio. It's not
supposed to use anything else if you use volumectl. If you use alsamixer and
volumectl at the same time, volumect will not give a shit about changed volume
in alsamixer, it will just change volume relatively to what was remembered.

Sometimes some shit happens with pulseaudio, that's why the tool is measuring
time on every single operation with pulseaudio, so it outputs logs of daemon
like this:

```
TIME [unpacking]                    0.49ms
TIME [pulse:set-sink-volume]        0.87ms
TIME [packing]                      0.18ms
TIME [serve connection]             1.91ms
TIME [unpacking]                    0.15ms
TIME [pulse:set-sink-volume]        0.88ms
TIME [packing]                      0.05ms
TIME [serve connection]             1.34ms
```

Volumectl doesn't limit you in your volume level, you can increase volume for
even more than 100%.

# How

## Run daemon
```
volumectl -D
```

## Do something with volume

Up volume for two percents:
```
 →  volumectl up 2
29.80
```

Down volume for three percents:
```
 →  volumectl down 3
26.80
```

Get current volume level:

```
 →  volumectl get -f '%.f'
27
```

# Also

Also, volumectl listens for dbus signals from pulseaudio, if you got connected
new sink or changed active sink port, volumectl will re-obtain information.

It happens in cases like:
- connect new device (for example, bluetooth speaker)
- remove device (for example, bluetooth speaker)
- connect wired headphones
- remove wired headphones

# Installation

## Poor man' way:

Just go-get the source code and find your binary in $GOPATH/bin:

```
go get github.com/kovetskiy/volumectl
```

Copy systemd service:

```
cp $GOPATH/src/github.com/kovetskiy/volumectl/systemd/volumectl.service /usr/lib/systemd/user/volumectl.service
```

Enable and start systemd service:

```
systemctl enable --user --now volumectl.service
```

## Arch Linux way

[Install from AUR: volumectl-git](https://aur.archlinux.org/packages/volumectl-git)

```
systemctl enable --user --now volumectl.service
```

## Configuration

Zero configuration, just run.

## License

MIT
