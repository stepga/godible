# go(au)dible

:-)

## Debugging/Infos

* Kernel info
  * `uname -r`
  * config `gzip -d /proc/config.gz -c  | less`
  * kernel parameters: `cat /proc/cmdline`

## TODOs

* save player state in /perm/godible-data/state.json
  * survives reboot
  * re-save once a minute
  * embed _recalculate_ (or something similar) button in site to generate a sqlite db with all elements

* save track json in /perm/godible-data/tracks.json
  * if existent upon start: load this json instead of creating the track list again

* do not use golang templating but print the table via js
  * get tbodies[] data periodically from API
  * generate tbody html via template literals for each tbody data (see https://stackoverflow.com/questions/66246946/how-to-fill-html-with-local-json-data)
  * add hash to each tbody https://stackoverflow.com/a/7616484
  * if tbody id not ex. or hash differs -> replace

* web interface
  * table format for track
    * add onclick events for basename to play the tracks (killer feature ;-))
    * bonus: column with centered action buttons as: play, enqueue, delete ... further future features :-)
  * upload songs
    * update player's internal file list
  * delete songs
    * update player's internal file list
  * play queue
    * add action buttons to enqueue tracks
    * needs an interactive queue overview

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

* implement websocket ping/ping as in https://github.com/gorilla/websocket/blob/main/examples/chat/home.html

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
