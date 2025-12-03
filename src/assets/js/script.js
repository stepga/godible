var time_current_lock = false;

var is_playing = false;

let play_svg;
let pause_svg;

function secondsToDateStr(seconds) {
	var date = new Date(null);
	date.setSeconds(seconds);
	if (seconds < 60*60) {
		return date.toISOString().slice(14, 19);
	}
	return date.toISOString().slice(11, 19);
}

function dateStrToSeconds(date) {
    var arr = date.split(':');
    var ret = 0;
    var seconds_multiplier = 1;

    while (arr.length > 0) {
        ret += seconds_multiplier * parseInt(arr.pop(), 10);
        seconds_multiplier *= 60;
    }

    return ret;
}

function setToggleButton() {
	var toggle = document.getElementById("toggle");
	if (is_playing) {
		toggle.innerHTML = pause_svg;
	} else {
		toggle.innerHTML = play_svg;
	}
}

function updateUI(data) {
	if (data == null || data == "null") {
		console.log("updateUI: no data passed");
		return;
	}
	var json = JSON.parse(data);
	var title = document.getElementById("title");
	var time_current = document.getElementById("time_current");
	var time_total = document.getElementById("time_total");
	var slider = document.getElementById("slider");

	// TODO: update fields only if content really changed
	title.textContent = json.name;
	time_total.textContent = secondsToDateStr(json.duration);
	slider.max = json.duration;
	if (time_current_lock == false) {
		time_current.textContent = secondsToDateStr(json.duration_current);
		slider.value = json.duration_current;
	}

	is_playing = json.is_playing;
	setToggleButton();
}

function fetchGlobalSvg(url, obj) {
	fetch(url)
		.then( r => r.text() )
		.then( t => obj = t )
}

document.addEventListener("DOMContentLoaded", function(event) {
	fetch("img/Font_Awesome_5_regular_pause-circle.svg").then( r => r.text() ).then( t => pause_svg = t )
	fetch("img/Font_Awesome_5_regular_play-circle.svg").then( r => r.text() ).then( t => play_svg = t )

	var websocket = new WebSocket("ws://"+window.location.host+"/ws");
	websocket.onmessage = function(event) {
		updateUI(event.data);
	}
	websocket.onerror = function(event) {
		console.log("WebSocket error: " + event.data);
	}

	var slider = document.getElementById("slider");
	slider.oninput = function() {
		const time_current = document.getElementById("time_current");
		time_current.textContent = secondsToDateStr(slider.value);
	}

	function downEvent() {
		time_current_lock = true;
	}
	slider.onmousedown = downEvent
	slider.ontouchstart = downEvent

	function upEvent() {
		const time_current_date_str = document.getElementById("time_current").textContent;
		const time_current_seconds = dateStrToSeconds(time_current_date_str);
		time_current_lock = false;
		websocket.send('{ "command": "jump", "payload": "' + time_current_seconds + '"}');
	}
	slider.onmouseup = upEvent
	slider.ontouchend = upEvent

	document.getElementById("previous").addEventListener('click', function() {
		websocket.send('{ "command": "previous", "payload": ""}');
	});
	document.getElementById("toggle").addEventListener('click', function() {
		websocket.send('{ "command": "toggle", "payload": ""}');
		is_playing = !is_playing;
		setToggleButton();
	});
	document.getElementById("next").addEventListener('click', function() {
		websocket.send('{ "command": "next", "payload": ""}');
	});
});
