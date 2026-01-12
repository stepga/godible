# go(au)dible

:-)

## Develop & Live-Test

To quickly test,
* quit/stop the application on your gokrazy device (reserved ports etc.)
* cross compile the binary
* scp it (via breakglass) to your device, e.g.:
```
GOKDEV=10.0.0.225 # fill in the relevant IP address of your raspberry pi
make && \
  scp godible "$GOKDEV":/tmp && \
  ssh "$GOKDEV" "chmod +x /tmp/godible" && \
  ssh "$GOKDEV" "/tmp/godible"  | tee /tmp/xxx
```

## Debugging/Infos

* Kernel info
  * `uname -r`
  * config `gzip -d /proc/config.gz -c  | less`
  * kernel parameters: `cat /proc/cmdline`

## TODOs

* webgui: replace with easier bootstrap version?
* webgui: implement read-only fixed-width logfile view (textarea/div), tee-ing slog output
* webgui: table row click: play item

* rfid: also learn rfid uids for directories
  * on context switch: save state (track + position)
  * switch back: restore state

* save player state in /perm/godible-data/state.json
  * survives reboot
  * re-save once a minute
  * embed _recalculate_ (or something similar) button in site to generate a sqlite db with all elements

* save track json in /perm/godible-data/tracks.json
  * if existent upon start: load this json instead of creating the track list again

* web interface
  * table format for track
    * add onclick events for basename to play the tracks (killer feature ;-))
    * bonus: column with centered action buttons as: play, delete ... further future features :-)
  * upload songs
    * update player's internal file list
  * delete songs
    * update player's internal file list

* powersaving/tweak settings
  * change antenna gain ([1-7])
  * polling intervals ...

* move gokrazy web interface to 1080 with autoredirect to 1443 and a self-signed cert
   * "HTTPPORT": "1080"
   * "HTTPSPORT": "1443"
   * "UseTLS": "self-signed"
 * run new interface on 443 with self signed certificate (tls/ssl is needed for websocket (at least in some? browsers))
   * generate cert: https://go.dev/src/crypto/tls/generate_cert.go
   * redirect: https://stackoverflow.com/a/63590299
   * redirect: https://stackoverflow.com/questions/37536006/how-do-i-rewrite-redirect-from-http-to-https-in-go
   * redirect: https://gist.github.com/d-schmidt/587ceec34ce1334a5e60

* implement Previous() reset of currently played track
  * introduce a (percentage) threshold of the file being played
  * when this threshold is not yet reached, change the track
  * otherwise: reset the current track's offset.

* gpio
  * implement long press functions
    * fast forward,
    * fast backword
  * simplifying buttons possible?
    * remove dedicated resistors between button and gnd
    * direct connection between gnd && button
    * use internal pullup resistor
    * if pin is low -> button is pressed
    * XXX: is debouncing needed (?)
    * `err := pinIO.In(gpio.PullUp, gpio.FallingEdge)` ?

* fritzing of the hardware setup

* jack plug (klinkenstecker) via gpio (the raspberry pi zero 2w does not have a dedicated phone connector)
  * https://learn.adafruit.com/adding-basic-audio-ouput-to-raspberry-pi-zero?view=all
  * https://raspberrypi.stackexchange.com/questions/49600/how-to-output-audio-signals-through-gpio
  * https://wiki.batocera.org/audio_via_gpio_rpi_only

## further feature ideas

* implement websocket ping/pong as in https://github.com/gorilla/websocket/blob/main/examples/chat/home.html

* implement recursive file/dir watch and update Player.audioSourceList
  * e.g via https://github.com/fsnotify/fsnotify/issues/18#issuecomment-3109424560

* usb webcam qr code module
  * decide: via button push or e.g. one webcam shot per second check?
  * see also https://github.com/makiuchi-d/gozxing

* active boxes: amazn.so/RafE4l8

* flip switch for power off/on
  * https://lowpowerlab.com/guide/atxraspi/full-pi-poweroff-from-software/
  * on/off shim
    * on pimorino itself https://shop.pimoroni.com/products/onoff-shim?variant=41102600138 (10EUR incl. tax)
    * on reichelt 9EUR
    * more soldering
    * huge bash installer script needed for raspbian ... further checking out needed how this actually works
  * self soldered flip-switch in usb cable between powerbank and raspberry?
    * hard power cut sucks though
    * https://www.instructables.com/OnOff-switch-for-a-USB-Powered-Device/

* run hostapd on long button push to create open wifi and to change wifi parameters data
  * hostapd (& dnsmasq?) in alpine-chroot (see [wrong] instructions: https://github.com/gokrazy/gokrazy/issues/49#issuecomment-105980013)
    ```
    # >>> start on host

    % cd /tmp
    % wget https://dl-cdn.alpinelinux.org/alpine/v3.23/releases/aarch64/alpine-minirootfs-3.23.0-aarch64.tar.gz
    % tar cf alpine.tar alpine-minirootfs-3.23.0-aarch64.tar.gz
    % go install github.com/gokrazy/breakglass/cmd/breakglass@latest
    % $GOK_INSTANCE=hello
    % breakglass -debug_tarball_pattern alpine.tar $GOK_INSTANCE

    # >>> breakglass is a ssh wrapper to the gokrazy instance
    # tar xf alpine-minirootfs-3.23.0-aarch64.tar.gz
    # mount -o bind /dev dev
    # chroot .

    # >>> whithin chroot on gokrazy instance
    / # mount -o proc proc /proc
    / # mount -o sysfs sys /sys
    / # echo nameserver 8.8.8.8 > /etc/resolv.conf
    / # apk add hostapd
    ```
  * build docker image with arm64 hostapd & dnsmasq (alpine base)?
    * Dockerfile could be part of repo (with a `make` target that build the image)
    * built image can be shipped via `ExtraFilePaths` (see `config.json` of the gokrazy instance's repo)
