/*
 * time_current_lock is needed to prevent updateUI writing time_current during
 * a currently happening interaction with the player's slider
 */
var time_current_lock = false;
/* is_playing is needed to set the correct play/pause icon */
var is_playing = false;

const createRowHTML = ({
	basename,
	current_seconds,
	duration_seconds,
	fullpath,
	fullpath_hash_sum,
	rfid_uid,
	hash_sum,

}) => `
<tr id="${fullpath_hash_sum}"
  data-basename="${basename}"
  data-fullpath="${fullpath}"
  data-hash_sum="${hash_sum}">
  <td>${basename}</td>
  <td class="text-center">${current_seconds} / ${duration_seconds}</td>
  <td class="text-center">
    <button
      id="rfid_button_${fullpath_hash_sum}"
      class="btn btn-warning mb-1"
      type="button">
      <i class="fa fa-wifi">
      ${rfid_uid}
      </i>
    </button>
  </td>
</tr>`;


// ----------
// TODO order
// ----------
// 1. backend should send via 'state' whether rfidlearning is in progess (incl. details)
//    - track name
//    - seconds left
// 2. frontend displays info in alertBox (no timeouts & further locking needed)
//    - while learning new track, disable all rfid buttons

function initializePlayerUI() {
	let slider = $("#slider");
	slider.on("input", function() {
		$("#time_current").text(secondsToHHMMSS(slider.val()));
	});

	function sliderDownEvent() {
	        time_current_lock = true;
	}
	slider.on("mousedown", sliderDownEvent);
	slider.on("touchstart", sliderDownEvent);

	function sliderUpEvent() {
		const time_current_date_str = $("#time_current").text();
		const time_current_seconds = HHMMSSToSeconds(time_current_date_str);
		time_current_lock = false;
		websocket.send('{ "type": "slide", "payload": "' + time_current_seconds + '"}');
	}
	slider.on("mouseup", sliderUpEvent);
	slider.on("touchend", sliderUpEvent);

	$("#previous").on("click", function() {
		websocket.send('{ "type": "previous", "payload": ""}');
	});
	$("#next").on("click", function() {
		websocket.send('{ "type": "next", "payload": ""}');
	});
	$("#toggle").on("click", function() {
		websocket.send('{ "type": "toggle", "payload": ""}');
		is_playing = !is_playing;
		setToggleIcon();
	});
	// TODO: implement me
	//document.getElementById("alertBoxCloseBtn").addEventListener('click', function() {
	//        hideAlertBox(true, "");
	//});
}

function HHMMSSToSeconds(date) {
    let arr = date.split(':');
    let ret = 0;
    let seconds_multiplier = 1;

    while (arr.length > 0) {
        ret += seconds_multiplier * parseInt(arr.pop(), 10);
        seconds_multiplier *= 60;
    }
    return ret;
}

function secondsToHHMMSS(seconds) {
	let date = new Date(null);
	date.setSeconds(seconds);
	// return MM:SS format if less than one hour
	if (seconds < 60*60) {
		return date.toISOString().slice(14, 19);
	}
	// otherwise return HH:MM:SS format
	return date.toISOString().slice(11, 19);
}

function setToggleIcon() {
	if (is_playing) {
		$("#toggleIcon").removeClass("fa-play");
		$("#toggleIcon").addClass("fa-pause");
	} else {
		$("#toggleIcon").removeClass("fa-pause");
		$("#toggleIcon").addClass("fa-play");
	}
}

function updateUI(data) {
	if (data == null || data == "null") {
		console.error("updateUI: no data passed");
		return;
	}

	let json;
	try {
		json = JSON.parse(data);
	} catch (e) {
		console.error("updateUI: " + e);
		return;
	}

	// TODO: update fields only if content really changed
	$("#track_name").text(json.name);
	$("#time_total").text(secondsToHHMMSS(json.duration));
	$("#slider").attr({ "max": json.duration });
	if (time_current_lock == false) {
		$("#time_current").text(secondsToHHMMSS(json.duration_current));
		$("#slider").val(json.duration_current);
	}

	is_playing = json.is_playing;
	setToggleIcon();

	// TODO: set alertBox if json.rfid_track_learn is not empty
}

function updateRfidButtonsClickEvent() {
	$('button[id*=rfid_]:not(.clickEventHandlerRegistered)').each(function(_index) {
		$(this).on("click", function() {
			const msg = JSON.stringify({
				type: "rfidtracklearn",
				payload: $(this).parent().parent().data('fullpath')
			});
			console.log('XXX send: ' + msg);
			websocket.send(msg);
		});
		$(this).addClass("clickEventHandlerRegistered");
	});
}

/* create a row's respective directory tbody, in which the row can be inserted */
function createRowTbody(rowStruct) {
	$(`<tbody
		id="${rowStruct['dirname_hash_sum']}"
		class="table-group-divider">
		<tr>
			<th colspan=2>${rowStruct['dirname']}</th>
			<td class="text-center">
			<button id="rfid_button_${rowStruct['dirname_hash_sum']}"
				class="btn btn-warning mb-1"
				type="button">
				<i class="fa fa-wifi"></i>
			</button>
			</td>
		</tr>
	</tbody>`).appendTo('table');
}

function updateTable(data) {
	if (data == null || data == "null") {
		console.error("updateTable: no data passed");
		return;
	}

	let json;
	try {
		json = JSON.parse(data);
	} catch (e) {
		console.error("updateTable: " + e);
		return
	}

	const rowsHTML = json.map(createRowHTML);
	for (let [index, rowHTML] of rowsHTML.entries()) {
		// update existing track rows
		var element = $("#" + json[index]['fullpath_hash_sum']);
		if (element.length !== 0) {
			if (element.data('hash_sum') != json[index]['hash_sum']) {
				element.replaceWith(rowHTML);
			}
			continue
		}

		// create directory tbodies if necessary
		var tbody = $("#" + json[index]['dirname_hash_sum']);
		if (tbody.length === 0) {
			createRowTbody(json[index]);
			tbody = $("#" + json[index]['dirname_hash_sum']);
		}

		// insert new track row
		$(rowHTML).appendTo(tbody);
	}
	updateRfidButtonsClickEvent();
}

function initializeWebsocket() {
	if (typeof(websocket) == 'undefined' || websocket == null) {
		console.log('initialize new websocket connection')
		websocket = new WebSocket("ws://"+window.location.host+"/ws");
	}
	websocket.onmessage = function(event) {
		let data = JSON.parse(event.data);
		switch (data['type']) {
			case "rows":
				updateTable(data['payload']);
				break;
			case "state":
				updateUI(data['payload']);
				break;
			case "hiderfidalertbox":
				// TODO: remove me and implement this also in 'state'
				break;
			default:
				console.error("websocket: unknown api request type '" + data['type'] + "'")
		}
	}
	websocket.onerror = function(event) {
		console.error("websocket error: " + event.data);
	}
}

function registerAlertBoxCloseButton() {
	$("#alertBoxCloseBtn").on("click", function() {
		$("#alertBox").toggle(false)
	});
}

function registerFilterSearch() {
	$("#filterInput").on("keyup", function() {
		let filterSearchString = $(this).val().toLowerCase();
		$("tbody tr").filter(function() {
			// directory of iterated row matches the search:
			// show all corresponding rows
			let directoryRow = $(this).parent().find('tr').eq(0);
			let directoryRowText = directoryRow.text().toLowerCase();
			if (directoryRowText.indexOf(filterSearchString) > -1) {
				$(this).toggle(true);
				return;
			}

			// iterated row (may also be the directory row) matches the search:
			// show row and its corresponding directory row
			let row = $(this);
			let rowText = row.text().toLowerCase();
			if (rowText.indexOf(filterSearchString) > -1) {
				$(this).toggle(true);
				directoryRow.toggle(true);
			} else {
				$(this).toggle(false);
			}
		});
	});
}

var websocket;

$(document).ready(function(){
	registerFilterSearch();
	registerAlertBoxCloseButton();
	initializeWebsocket();
	initializePlayerUI();
});
