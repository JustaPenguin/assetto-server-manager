"use strict";

let $document;

// entry-point
$(document).ready(function () {
    console.log("initialising server manager javascript");

    $document = $(document);

    tracks.init();
    serverLogs.init();
});

let tracks = {
    // jQuery elements
    $trackDropdown: null,
    $trackLayoutDropdown: null,
    $trackLayoutDropdownParent: null,

    // the current layout as specified by the server
    currentLayout: "",

    // all available track layout options
    trackLayoutOpts: {},

    // init: entrypoint for tracks functions. looks for track + layout dropdowns and populates them.
    init: function () {
        tracks.$trackDropdown = $document.find("#Track");
        tracks.$trackLayoutDropdown = $document.find("#TrackLayout");
        tracks.$trackLayoutDropdownParent = tracks.$trackLayoutDropdown.closest(".form-group");

        // restrict loading track layouts to pages which have track dropdown and layout dropdown on them.
        if (tracks.$trackDropdown.length && tracks.$trackLayoutDropdown.length) {
            // build a map of track => available layouts
            tracks.$trackLayoutDropdown.find("option").each(function (index, opt) {
                let $optValSplit = $(opt).val().split(":");
                let trackName = $optValSplit[0];
                let trackLayout = $optValSplit[1];

                if (!tracks.trackLayoutOpts[trackName]) {
                    tracks.trackLayoutOpts[trackName] = [];
                }

                tracks.trackLayoutOpts[trackName].push(trackLayout);

                if ($optValSplit.length > 2) {
                    tracks.currentLayout = trackLayout;
                }
            });

            tracks.$trackLayoutDropdownParent.hide();
            tracks.loadTrackLayouts();

            tracks.$trackDropdown.change(tracks.loadTrackLayouts);
        }
    },

    // loadTrackLayouts: looks at the selected track and loads in the correct layouts for it into the
    // track layout dropdown
    loadTrackLayouts: function () {
        tracks.$trackLayoutDropdown.empty();

        let selectedTrack = tracks.$trackDropdown.find("option:selected").val();
        let availableLayouts = tracks.trackLayoutOpts[selectedTrack];

        if (availableLayouts) {
            for (let i = 0; i < availableLayouts.length; i++) {
                tracks.$trackLayoutDropdown.append(tracks.buildTrackLayoutOption(availableLayouts[i]));
            }

            tracks.$trackLayoutDropdownParent.show();
        } else {
            // add an option with an empty value
            tracks.$trackLayoutDropdown.append(tracks.buildTrackLayoutOption(""));
            tracks.$trackLayoutDropdownParent.hide();
        }
    },

    // buildTrackLayoutOption: builds an <option> containing track layout information
    buildTrackLayoutOption: function (layout) {
        let $opt = $("<option/>");
        $opt.attr({'value': layout});
        $opt.text(layout);

        if (layout === tracks.currentLayout) {
            $opt.prop("selected", true);
        }

        return $opt;
    },
};


let serverLogs = {
    init: function() {
        let $serverLog = $document.find("#server-logs");
        let $managerLog = $document.find("#manager-logs");

        if ($serverLog.length && $managerLog.length) {

            setInterval(function() {
                $.get("/api/logs", function (data) {
                    $serverLog.text(data.ServerLog);
                    $managerLog.text(data.ManagerLog);
                });
            }, 1000);
        }
    },
};

const layout = {
    preview: "",
    details: "",
};

let filesToUpload = [];

function submitFiles() {
    //JSON encode filestoUpload, JQUERY post request to api endpoint (/api/track/upload)
    let newFiles = [];
    let count = 0;

    for (let x = 0; x < filesToUpload.length; x++) {
        // Set preview to base64 encoded image
        let reader = new FileReader();

        reader.readAsDataURL(filesToUpload[x]);

        reader.addEventListener("load", function () {
            newFiles.push({
                'name'                  : filesToUpload[x].name,
                'size'                  : filesToUpload[x].size,
                'type'                  : filesToUpload[x].type,
                'webkitRelativePath'    : filesToUpload[x].webkitRelativePath,
                'dataBase64'            : reader.result.toString()
            });

            count++;

            if (count === filesToUpload.length) {
                jQuery.post("/api/track/upload", JSON.stringify(newFiles), onSuccess).fail(onFail)
            }
        });
    }
}

function onSuccess(data) {
    console.log("Track Successfully Added");

    let $trackPanel = $("#track-info-panel");
    $trackPanel.attr({'class': "card p-3 mt-2"});
    $trackPanel.text("Track(s) Successfully Added");
}

function onFail(data) {
    console.log("Track Could Not be Added")
}

function handleFiles(fileList) {
    let layouts = {};
    let layoutNum = 0;
    let filesToUploadLocal = [];

    for (let x = 0; x < fileList.length; x++) {

        // get model/surfaces and drs zones and ui folder
        if ((fileList[x].name.startsWith("models") && fileList[x].name.endsWith(".ini")) ||
            (fileList[x].name === "surfaces.ini" || fileList[x].name === "drs_zones.ini") ||
            (fileList[x].webkitRelativePath.includes("/ui/"))) {

            filesToUploadLocal.push(fileList[x])
        }

        if (fileList[x].name.startsWith("models")) {
            layoutNum++
        }
    }

    let tableDone = false;
    let $trackPanel = $("#track-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");

    $trackPanel.attr({'class': "card p-3 mt-2"});
    $title.text("Track Preview");
    $row.attr({'class': "card-deck"});

    $trackPanel.append($title);

    $trackPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {
        if (filesToUploadLocal[x].webkitRelativePath.includes("/ui/")) {

            if (filesToUploadLocal[x].name === "preview.png") {

                let layoutName = "";

                if (layoutNum > 1) {
                    let fileListCorrected = filesToUploadLocal[x].webkitRelativePath.replace('\\', '/');

                    let fileListSplit = fileListCorrected.split("/");

                    layoutName = fileListSplit[2];
                } else {
                    layoutName = "Default";
                }

                if (!layouts[layoutName]) {
                    layouts[layoutName] = Object.create(layout);
                }

                // Set preview to base64 encoded image
                let reader = new FileReader();

                reader.readAsDataURL(filesToUploadLocal[x]);

                reader.addEventListener("load", function () {
                    layouts[layoutName].preview = reader.result.toString();

                    let layoutInfo = layouts[layoutName];

                    $row.append(buildTrackLayoutPanel(layoutInfo.preview, layoutName));
                });
            }

            if (filesToUploadLocal[x].name === "ui_track.json" && !tableDone) {
                tableDone = true;
                let reader = new FileReader();

                reader.readAsText(filesToUploadLocal[x]);

                reader.addEventListener("load", function () {
                    $trackPanel.append(buildHtmlTable([JSON.parse(reader.result.toString())]));
                });
            }
        }

    }

    let $uploadButton = $("#upload-button");
    $uploadButton.attr({'class': "d-inline"});

    if (filesToUploadLocal.length === 0) {
        $uploadButton.text("Sorry, the files you uploaded don't seem to be a compatible track!");
    } else {
        if (!$("#track-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({'class': "btn btn-primary", 'onclick': "submitFiles()", 'id': "track-upload-button"});
            $button.text("Upload Track(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function buildTrackLayoutPanel(img, layoutName) {
    let $panel = $("<div/>");
    let $img = $("<img/>");
    let $cardBody = $("<div/>");
    let $cardText = $("<h5/>");

    $img.attr({'src': img});
    $img.attr({'alt': "Track Preview"});
    $img.attr({'class': "card-img-top"});

    $cardBody.attr({'class': "card-body"});

    $cardText.attr({'class': "card-title"});
    $cardText.text("Layout: " + layoutName);

    $cardBody.append($cardText);

    $panel.append($img);
    $panel.append($cardBody);

    $panel.attr({'class': "card text-center mb-3"});

    return $panel;
}

// Builds the HTML Table out of myList json data from Ivy restful service.
function buildHtmlTable(json) {
    let $cardTable = $("<table/>");

    $cardTable.attr({'class': "table table-sm table-bordered"});
    $cardTable.attr({'id': "layout-table"});

    let columns = addAllColumnHeaders(json, $cardTable);

    for (let i = 0 ; i < json.length ; i++) {
        let $row = $('<tr/>');
        for (let colIndex = 0 ; colIndex < columns.length ; colIndex++) {
            let cellValue = json[i][columns[colIndex]] + "<br>";

            if (cellValue == null) { cellValue = ""; }

            $row.append($('<td/>').html(cellValue));
        }
        $cardTable.append($row);
    }

    return $cardTable
}

// Adds a header row to the table and returns the set of columns.
// Need to do union of keys from all records as some records may not contain
// all records
function addAllColumnHeaders(json, table)
{
    let columnSet = [];
    let headerTr$ = $('<tr/>');
    let header$ = $('<thead/>');

    header$.attr({'class': "thead-dark"});

    for (let i = 0 ; i < json.length ; i++) {
        let rowHash = json[i];
        for (let key in rowHash) {
            if ($.inArray(key, columnSet) == -1){
                if (key === "tags" || key === "run" || key === "url") {
                    continue
                }

                columnSet.push(key);
                headerTr$.append($('<th/>').html(key));
            }
        }
    }

    header$.append(headerTr$);

    table.append(header$);

    return columnSet;
}