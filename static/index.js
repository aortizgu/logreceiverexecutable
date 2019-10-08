var LIVE_DATA = [];
var SEARCH_DATA = [];
var EXPORT_DATA = [];

var LIVE_FILTERS = {};
LIVE_FILTERS["severity"] = "";
LIVE_FILTERS["hostname"] = "";
LIVE_FILTERS["device"] = "";
var MAX_ENTRIES = 25000;

var scroll_auto = false;
var live_view = true;

var waitingDialog = waitingDialog || (function ($) {
    'use strict';

	// Creating modal dialog's DOM
	var $dialog = $(
		'<div class="modal fade" data-backdrop="static" data-keyboard="false" tabindex="-1" role="dialog" aria-hidden="true" style="padding-top:15%; overflow-y:visible;">' +
		'<div class="modal-dialog modal-m">' +
		'<div class="modal-content">' +
			'<div class="modal-header"><h3 style="margin:0;"></h3></div>' +
			'<div class="modal-body">' +
				'<div class="progress progress-striped active" style="margin-bottom:0;"><div class="progress-bar" style="width: 100%"></div></div>' +
			'</div>' +
		'</div></div></div>');

	return {
		/**
		 * Opens our dialog
		 * @param message Custom message
		 * @param options Custom options:
		 * 				  options.dialogSize - bootstrap postfix for dialog size, e.g. "sm", "m";
		 * 				  options.progressType - bootstrap postfix for progress bar type, e.g. "success", "warning".
		 */
		show: function (message, options) {
			// Assigning defaults
			if (typeof options === 'undefined') {
				options = {};
			}
			if (typeof message === 'undefined') {
				message = 'Loading';
			}
			var settings = $.extend({
				dialogSize: 'm',
				progressType: '',
				onHide: null // This callback runs after the dialog was hidden
			}, options);

			// Configuring dialog
			$dialog.find('.modal-dialog').attr('class', 'modal-dialog').addClass('modal-' + settings.dialogSize);
			$dialog.find('.progress-bar').attr('class', 'progress-bar');
			if (settings.progressType) {
				$dialog.find('.progress-bar').addClass('progress-bar-' + settings.progressType);
			}
			$dialog.find('h3').text(message);
			// Adding callbacks
			if (typeof settings.onHide === 'function') {
				$dialog.off('hidden.bs.modal').on('hidden.bs.modal', function (e) {
					settings.onHide.call($dialog);
				});
			}
			// Opening dialog
			$dialog.modal();
		},
		/**
		 * Closes dialog
		 */
		hide: function () {
			$dialog.modal('hide');
		}
	};

})(jQuery);


function connectionLost() {
	$("#no-connection").show();
}

function connectionOK() {
	$("#no-connection").hide();
}

function showLive() {
	live_view = true;
	$('#console_logs_live_div').show();
	$('#console_logs_search_div').hide();
}

function showSearch() {
	live_view = false;
	$('#console_logs_live_div').hide();
	$('#console_logs_search_div').show();
}

function exportCSV() {
	if($('#console_logs_live_div').is(":visible")){
		var toExport = [];
		var log = undefined;
		for ( var i in LIVE_DATA) {
			log = LIVE_DATA[i]
			if(matchsLiveFilter(log)){
				toExport.push(log);
			}
		}
		var csvData = toCsv(toExport);
		download(csvData, 'log_export.csv', 'text/json');
	}else{
		var csvData = toCsv(SEARCH_DATA);
		download(csvData, 'log_export.csv', 'text/json');
	}
}

function exportTXT() {
	if($('#console_logs_live_div').is(":visible")){
		var txtData = toTxt(LIVE_DATA);
		download(txtData, 'log_export.txt', 'text/json');
	}else{
		var txtData = toTxt(SEARCH_DATA);
		download(txtData, 'log_export.txt', 'text/json');
	}
}

function matchsLiveFilter(log){
	var match_severity = true;
	var match_hostname = true;
	var match_device = true;
	if(LIVE_FILTERS["severity"].length > 0){
		match_severity = String(log["severity"]).toLowerCase().includes(LIVE_FILTERS["severity"]);
	}
	if(LIVE_FILTERS["hostname"].length > 0){
		match_hostname = log["hostname"].toLowerCase().includes(LIVE_FILTERS["hostname"]);
	}
	if(LIVE_FILTERS["device"].length > 0){
		match_device = log["app"].toLowerCase().includes(LIVE_FILTERS["device"]);
	}
	return match_severity && match_hostname && match_device;
}

function newLiveData(data) {
	console.log("new data");
	var log = JSON.parse(data);
	LIVE_DATA.push(log);
	if(matchsLiveFilter(log)){
		var row = "<tr><td>" + log['severity'] + "</td><td>" + new Date(log['timestamp']*1000).toLocaleString() + "</td><td>" + log['hostname'] + "</td><td>"  + log['app_name'] + "</td><td>"  + log['message'] + "</td></tr>";
		$('#console_logs_live').append(row);
		if(scroll_auto && live_view){
			window.scrollTo(0,document.body.scrollHeight);		
		}
	}
}

function applyLiveFilters() {	
	$('#console_logs_live').html("");
	for ( var i in LIVE_DATA) {
		log = LIVE_DATA[i];
		if(matchsLiveFilter(log)){
			var row = "<tr><td>" + log['severity'] + "</td><td>" + new Date(log['timestamp']*1000).toLocaleString() + "</td><td>" + log['hostname'] + "</td><td>"  + log['app_name'] + "</td><td>"  + log['message'] + "</td></tr>";
			$('#console_logs_live').append(row);
		}
	}
}

function getInfo() {
	$.ajax({
		url : "/info",
		type : 'GET',
		cache : false
	}).done(function(info, status) {
		$('#dialog_about_version').html(
				"Version: " + info['version'] + "<br>" + 
				"Port: " + info['port'] + "<br>" +
				"Max logs to store: " + info['max_logs'] + "<br>" +
				"Logs stored: " + info['log_count'] + "<br>" +
				"Open web sockets: " + info['open_web_sockets_count'] + "<br>" +
				"Uptime: " + info['uptime'] + "<br>");
	}).fail(function() {
		console.log("error loading " + resource);
	}).always(function() {
	});
}

function searchLogs(device, hostname, severity, day, maxEntries, offsetEntries){
	var resource = "/search?device=" + device + "&hostname=" + hostname + "&severity=" + severity + "&day=" + day + "&max=" + maxEntries + "&offset=" + offsetEntries;
	$.ajax({
		url : resource,
		type : 'GET',
		cache : false,
	}).done(function(data, status) {
		logs = data;
		for ( var i in logs) {
			log = logs[i];
			SEARCH_DATA.push(log);
			var row = "<tr><td>" + log['severity'] + "</td><td>" + new Date(log['timestamp']*1000).toLocaleString() + "</td><td>" + log['hostname'] + "</td><td>"  + log['app_name'] + "</td><td>"  + log['message'] + "</td></tr>";
			$('#console_logs_search').append(row);
		}
		if(logs.length == 0 || logs.length != maxEntries){
			waitingDialog.hide();
		}else{
			waitingDialog.show('Downloading logs, ' + (offsetEntries + maxEntries) + " downloaded.");
			searchLogs(device, hostname, severity, day, maxEntries, offsetEntries + maxEntries);
		}
	}).fail(function() {
		console.log("error loading " + resource);
		waitingDialog.hide();
	}).always(function() {
	});
}

function exportLogs(from_str, device, hostname, severity, day, maxEntries, offsetEntries){
	var resource = "/search?device=" + device + "&hostname=" + hostname + "&severity=" + severity + "&day=" + day + "&max=" + maxEntries + "&offset=" + offsetEntries;
	$.ajax({
		url : resource,
		type : 'GET',
		cache : false
	}).done(function(data, status) {
		logs = data;
		for ( var i in logs) {
			log = logs[i];
			EXPORT_DATA.push(log);
		}
		if(logs.length == 0 || logs.length != maxEntries){
			var csvData = toCsv(EXPORT_DATA);
			download(csvData, 'log_export_' + from_str + '.csv', 'text/json');
			waitingDialog.hide();
		}else{
			waitingDialog.show('Downloading logs, ' + (offsetEntries + maxEntries) + " downloaded.");
			exportLogs(from_str, device, hostname, severity, day, maxEntries, offsetEntries + maxEntries);
		}
	}).fail(function() {
		console.log("error loading " + resource);
		waitingDialog.hide();
	}).always(function() {
	});
}

//ON LOAD
$('document').ready(function() {
	WebSocketInit();

	$('#live_view_button').click(function(e) {
		showLive();
		e.preventDefault();
	});

	$('#search_view_button').click(function(e) {
		showSearch();
		e.preventDefault();
	});

	$('#enable_scroll_button').click(function(e) {
		scroll_auto = true;
		e.preventDefault();
	});

	$('#disable_scroll_button').click(function(e) {
		scroll_auto = false;
		e.preventDefault();
	});

	$('#about_button').click(function(e) {
		getInfo();
		$("#dialog_about").modal()
	});

	$('#live_filters').click(function () {
		$("#dialog_filter_live").modal()
	});

	$('#search').click(function () {
		$("#dialog_filter_search").modal()

	});
	
	$('#export_live').click(function () {
		$("#dialog_filter_export").modal()
	});
	
	$('#export_search').click(function () {
		$("#dialog_filter_export").modal()
	});
	
	$('#export_csv_button').click(function () {
		exportCSV();
	});

	$('#export_txt_button').click(function () {
		exportTXT();
	});

	$('.clendar-search').change(function() {
		SEARCH_ROLE = true;
	});

	$('#ok_live_filters').click(function (e) {
		var severity = $('#dialog_filter_severity')[0].value.toLowerCase();
		var hostname = $('#dialog_filter_hostname')[0].value.toLowerCase();
		var device = $('#dialog_filter_device')[0].value.toLowerCase();
		LIVE_FILTERS["severity"] = severity.toLowerCase();
		LIVE_FILTERS["hostname"] = hostname.toLowerCase();
		LIVE_FILTERS["device"] = device.toLowerCase();
		applyLiveFilters();
		e.preventDefault();		
	});

	$('#clear_live_filters').click(function (e) {
		LIVE_FILTERS["severity"] = "";
		LIVE_FILTERS["hostname"] = "";
		LIVE_FILTERS["device"] = "";
		applyLiveFilters();
		e.preventDefault();		
	});

	$('#clear_live_logs').click(function (e) {
		LIVE_DATA = [];
		$('#console_logs_live').html("");
		e.preventDefault();		
	});

	$('#clear_search').click(function (e) {
		$('#console_logs_search').html("");
		SEARCH_DATA = [];
		e.preventDefault();		
	});

	$('#ok_search_filters').click(function (e) {
		var device = $('#dialog_search_device')[0].value.toLowerCase();
		var hostname = $('#dialog_search_hostname')[0].value.toLowerCase();
		var severity = $('#dialog_search_severity')[0].value.toLowerCase();
		var day = $("#day-search").val();
		waitingDialog.show('Downloading logs...');
		SEARCH_DATA = [];
		$('#console_logs_search').html("");
		searchLogs(device, hostname, severity, day, MAX_ENTRIES, 0);
		e.preventDefault();		
	});

	$('#ok_export_filters').click(function (e) {
		var device = $('#dialog_export_device')[0].value.toLowerCase();
		var hostname = $('#dialog_export_hostname')[0].value.toLowerCase();
		var severity = $('#dialog_export_severity')[0].value.toLowerCase();
		var day = $("#day-export").val();
		waitingDialog.show('Downloading logs...');
		EXPORT_DATA = [];
		exportLogs(day, device, hostname, severity, day, MAX_ENTRIES, 0);
		e.preventDefault();		
	});

	$('.day').datepicker({
		format: 'dd-mm-yyyy',
		autoclose: true,
		todayHighlight: true
	});

	var date = new Date();
	var today = new Date(date.getFullYear(), date.getMonth(), date.getDate());
	$('.day').datepicker( 'setDate', today );
	showLive();
})
