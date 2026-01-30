function initializeWebsocket(websocket) {
	if (typeof(websocket) == 'undefined' || websocket == null) {
		console.log('initialize new websocket connection')
		websocket = new WebSocket("ws://"+window.location.host+"/ws");
	}
	websocket.onmessage = function(event) {
		let data = JSON.parse(event.data);
		switch (data['type']) {
			case "state":
				// TODO: implement me
				//updateUI(data['payload']);
				break;
			case "rows":
				// TODO: implement me
				//updateTable(data['payload']);
				break;
			case "hiderfidalertbox":
				// TODO: implement me
				//hideAlertBox(true, "");
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

$(document).ready(function(){
	registerFilterSearch();
	registerAlertBoxCloseButton();
	// TODO:
	// - slider
	// - table tbody hiding
	var websocket;
	initializeWebsocket(websocket);
});
