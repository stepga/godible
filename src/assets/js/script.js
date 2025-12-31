var time_current_lock = false;
var is_playing = false;
var play_svg;
var pause_svg;

function secondsToDateStr(seconds) {
	let date = new Date(null);

	date.setSeconds(seconds);

	if (seconds < 60*60) {
		return date.toISOString().slice(14, 19);
	}
	return date.toISOString().slice(11, 19);
}

function dateStrToSeconds(date) {
    let arr = date.split(':');
    let ret = 0;
    let seconds_multiplier = 1;

    while (arr.length > 0) {
        ret += seconds_multiplier * parseInt(arr.pop(), 10);
        seconds_multiplier *= 60;
    }

    return ret;
}

function setToggleButton() {
	let toggle = document.getElementById("toggle");
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
	let json = JSON.parse(data);
	let title = document.getElementById("title");
	let time_current = document.getElementById("time_current");
	let time_total = document.getElementById("time_total");
	let slider = document.getElementById("slider");

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

function tbodiesWithDirnamePrefix(dirname) {
	let ret = [];
	let ths = document.querySelectorAll('tbody tr th');

	for (let th of ths) {
		if (th.textContent.startsWith(dirname)) {
			ret.push(th.parentElement.parentElement)
		}
	}
	return ret;
}

function tbodyIsExpanded(tbody) {
	return tbody.querySelector('tr th').classList.contains('fa-folder-o')
}

function tbodyExpand(tbody, doExpand) {
	if (doExpand) {
		tbody.querySelector("th").classList.replace("fa-folder", "fa-folder-o");
	} else {
		tbody.querySelector("th").classList.replace("fa-folder-o", "fa-folder");
	}
	for (let td of tbody.getElementsByTagName("td")) {
		td.style.display = doExpand ? "" : "none";
	}
}

var websocket;

document.addEventListener("DOMContentLoaded", function(event) {
	fetch("img/Font_Awesome_5_regular_pause-circle.svg").then( r => r.text() ).then( t => pause_svg = t )
	fetch("img/Font_Awesome_5_regular_play-circle.svg").then( r => r.text() ).then( t => play_svg = t )

	let table = document.getElementsByTagName("table")[0];
	table.addEventListener("click", function(event){
		let elem = event.target;
		const classNames = ['fa-folder', 'fa-folder-o', 'folder'];
		if (classNames.some(className => elem.classList.contains(className)) == false) {
			return;
		}
		let doExpand = !tbodyIsExpanded(elem.parentElement.parentElement);
		let tbodies = tbodiesWithDirnamePrefix(elem.textContent);
		for (let tbody of tbodies) {
			tbodyExpand(tbody, doExpand);
		}

		for (let tbody of document.querySelectorAll('tbody')) {
			if (tbodyIsExpanded(tbody)) {
				document.querySelector('thead tr').style.opacity = "";
				return;
			}
		}
		document.querySelector('thead tr').style.opacity = "0.3";
	});

	if (typeof(websocket) == 'undefined' || websocket == null) {
		console.log('initialize websocket connection')
		websocket = new WebSocket("ws://"+window.location.host+"/ws");
	}
	websocket.onmessage = function(event) {
		let data = JSON.parse(event.data);
		switch (data['type']) {
		case "state":
			updateUI(data['payload']);
			break;
		case "rows":
			updateTable(data['payload']);
			break;
		default:
			console.log("unknown websocket api request type: " + data['type'])
		}
	}
	websocket.onerror = function(event) {
		console.log("WebSocket error: " + event.data);
	}

	let slider = document.getElementById("slider");
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
		websocket.send('{ "type": "slide", "payload": "' + time_current_seconds + '"}');
	}
	slider.onmouseup = upEvent
	slider.ontouchend = upEvent

	document.getElementById("previous").addEventListener('click', function() {
		websocket.send('{ "type": "previous", "payload": ""}');
	});
	document.getElementById("toggle").addEventListener('click', function() {
		websocket.send('{ "type": "toggle", "payload": ""}');
		is_playing = !is_playing;
		setToggleButton();
	});
	document.getElementById("next").addEventListener('click', function() {
		websocket.send('{ "type": "next", "payload": ""}');
	});
	document.getElementById("alertBoxCloseBtn").addEventListener('click', function() {
		hideAlertBox(true, "");
	});
});

function updateTable(data) {
	if (data == null || data == "null") {
		console.log("updateTable: no data passed");
		return;
	}
	let json = JSON.parse(data);

	const rows = json.map(createRowHTML);
	for (let [index, row] of rows.entries()) {
		var element = document.getElementById(json[index]['fullpath']);
		if (typeof(element) != 'undefined' && element != null) {
			continue;
		}
		let dirname = json[index]['dirname'];
		const tbody = getTBodyWithDirname(dirname)
		tbody.innerHTML += row;
	}
	updateRfidButtonEvents()
}

function updateRfidButtonEvents() {
	for (let rfidBtn of document.querySelectorAll('button[class~="fa-wifi"]')) {
		eventlistenerunset = rfidBtn.dataset['eventlistenerunset']
		if (typeof(eventlistenerunset) == 'undefined' || eventlistenerunset == null) {
			// eventlistener already added and data attribute remove
			continue
		}
		rfidBtn.addEventListener("click", function(event) {
			hideAlertBox(false, rfidBtn.dataset.basename);

			websocket.send('{ "type": "rfidtracklearn", "payload": "' + rfidBtn.dataset['fullpath'] + '"}');

			//ticker function that will refresh our display every second
			var date_old = new Date();
			const max_sec = 10;
			var refreshIntervalId = setInterval(function() {
				var date_new = new Date();
				var over_sec = Math.floor((date_new - date_old)/1000);
				var cur_sec = max_sec - over_sec;
				document.getElementById('alertBoxSeconds').textContent = cur_sec;
				if (cur_sec < 0) {
					clearInterval(refreshIntervalId);
					hideAlertBox(true, "");
				}
			}, 1000);
		});
		rfidBtn.removeAttribute('data-eventlistenerunset');
	};
}

function getTBodyWithDirname(dirname) {
	for (let th of document.querySelectorAll('tbody > tr > th:nth-child(1)')) {
		if (th.textContent == dirname) {
			return th.parentElement.parentElement;
		}
	}
	const tbody = document.createElement("tbody");
	tbody.innerHTML = `
<tr>
	<th class="folder fa-folder-o" colspan="1" scope="rowgroup">${dirname}</th>
	<th class="folder buttons" scope="rowgroup"> </th>
	<th class="folder buttons" scope="rowgroup">
	</th>
</tr>`;
	document.querySelector('table').appendChild(tbody);
	return tbody;
}

function hideAlertBox(hide, text) {
	alertBoxTrackName = document.getElementById("alertBoxTrackName");
	alertBoxTrackName.textContent = text;

	alertbox = document.getElementById("alertBox");
	if (hide) {
		alertbox.style.display='none';
	} else {
		alertbox.style.display='block';
	}
}

/**
 * Create one big string with interpolated values
 */
const createRowHTML = ({
	basename,
	current_seconds,
	dirname,
	duration_seconds,
	fullpath,
	rfid_uid,
}) => `
<tr id="${fullpath}">
  <td>${basename}</td>
  <td>${current_seconds} / ${duration_seconds}</td>
  <td class="buttons">
    <button data-eventlistenerunset="true" data-basename="${basename}" data-fullpath="${fullpath}" class="fa-wifi"> ${rfid_uid}</button>
  </td>
</tr>`;
