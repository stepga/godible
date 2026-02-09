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
// 1. updateUI
// 2. slider
// 3. rfid

function initializePlayerUI() {
	// TODO: implement me

	//let slider = document.getElementById("slider");
	//slider.oninput = function() {
	//        const time_current = document.getElementById("time_current");
	//        time_current.textContent = secondsToDateStr(slider.value);
	//}

	//function downEvent() {
	//        time_current_lock = true;
	//}
	//slider.onmousedown = downEvent
	//slider.ontouchstart = downEvent

	//function upEvent() {
	//        const time_current_date_str = document.getElementById("time_current").textContent;
	//        const time_current_seconds = dateStrToSeconds(time_current_date_str);
	//        time_current_lock = false;
	//        websocket.send('{ "type": "slide", "payload": "' + time_current_seconds + '"}');
	//}
	//slider.onmouseup = upEvent
	//slider.ontouchend = upEvent

	//document.getElementById("previous").addEventListener('click', function() {
	//        websocket.send('{ "type": "previous", "payload": ""}');
	//});
	//document.getElementById("toggle").addEventListener('click', function() {
	//        websocket.send('{ "type": "toggle", "payload": ""}');
	//        is_playing = !is_playing;
	//        setToggleButton();
	//});
	//document.getElementById("next").addEventListener('click', function() {
	//        websocket.send('{ "type": "next", "payload": ""}');
	//});
	//document.getElementById("alertBoxCloseBtn").addEventListener('click', function() {
	//        hideAlertBox(true, "");
	//});
}


function updateUI(data) {
	// TODO: implement me

	//if (data == null || data == "null") {
	//	//console.log("updateUI: no data passed");
	//	return;
	//}
	//let json = JSON.parse(data);
	//let title = document.getElementById("title");
	//let time_current = document.getElementById("time_current");
	//let time_total = document.getElementById("time_total");
	//let slider = document.getElementById("slider");

	//// TODO: update fields only if content really changed
	//title.textContent = json.name;
	//time_total.textContent = secondsToDateStr(json.duration);
	//slider.max = json.duration;
	//if (time_current_lock == false) {
	//	time_current.textContent = secondsToDateStr(json.duration_current);
	//	slider.value = json.duration_current;
	//}

	//is_playing = json.is_playing;
	//setToggleButton();
}

function updateRfidButtonEvents() {
	// TODO: implement me

	// - iterate over all buttons
	// - set click event if unset
	// - function:
	//   - websocket send rfidtracklearn with fullpath payload
	//   - backend must answer alertbox info (via websocket 'state'?): track name, seconds left, etc
	//     -> display alert box in frontend
	//     -> disable other rfid buttons
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
	updateRfidButtonEvents();
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
				// TODO: implement me
				//updateUI(data['payload']);
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
