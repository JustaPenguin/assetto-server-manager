"use strict";

let $document;

// entry-point
$(document).ready(function () {
    console.log("initialising server manager javascript");

    $document = $(document);

    // init bootstrap-switch
    $.fn.bootstrapSwitch.defaults.size = 'small';
    $.fn.bootstrapSwitch.defaults.animate = false;
    $.fn.bootstrapSwitch.defaults.onColor = "success";
    $document.find("input[type='checkbox']").bootstrapSwitch();

    raceSetup.init();
    serverLogs.init();
});

let raceSetup = {
    // jQuery elements
    $trackDropdown: null,
    $trackLayoutDropdown: null,
    $trackLayoutDropdownParent: null,
    $carsDropdown: null,
    $tyresDropdown: null,
    $addWeatherButton: null,

    // the current layout as specified by the server
    currentLayout: "",

    // all available track layout options
    trackLayoutOpts: {},


    /**
     * init: entrypoint for raceSetup functions. looks for track + layout dropdowns and populates them.
     */
    init: function () {
        raceSetup.$carsDropdown = $document.find("#Cars");

        raceSetup.$trackDropdown = $document.find("#Track");
        raceSetup.$trackLayoutDropdown = $document.find("#TrackLayout");
        raceSetup.$trackLayoutDropdownParent = raceSetup.$trackLayoutDropdown.closest(".form-group");

        raceSetup.$addWeatherButton = $document.find("#addWeather");

        if (raceSetup.$carsDropdown) {
            raceSetup.$carsDropdown.multiSelect();
            raceSetup.$tyresDropdown = $document.find("#LegalTyres");

            if (raceSetup.$tyresDropdown) {
                raceSetup.$tyresDropdown.multiSelect();

                raceSetup.$carsDropdown.change(raceSetup.populateTyreDropdown)
            }
        }

        raceSetup.$addWeatherButton.click(raceSetup.addWeather);

        $document.find(".weather-graphics").change(function() {
            let $this = $(this);

            $this.parent().parent().find(".weather-preview").attr({
                'src': '/content/weather/' + $this.val() + '/preview.jpg',
                'alt': $this.val(),
            });
        });

        // restrict loading track layouts to pages which have track dropdown and layout dropdown on them.
        if (raceSetup.$trackDropdown.length && raceSetup.$trackLayoutDropdown.length) {
            // build a map of track => available layouts
            raceSetup.$trackLayoutDropdown.find("option").each(function (index, opt) {
                let $optValSplit = $(opt).val().split(":");
                let trackName = $optValSplit[0];
                let trackLayout = $optValSplit[1];

                if (!raceSetup.trackLayoutOpts[trackName]) {
                    raceSetup.trackLayoutOpts[trackName] = [];
                }

                raceSetup.trackLayoutOpts[trackName].push(trackLayout);

                if ($optValSplit.length > 2) {
                    raceSetup.currentLayout = trackLayout;
                }
            });

            raceSetup.$trackLayoutDropdownParent.hide();
            raceSetup.loadTrackLayouts();

            raceSetup.$trackDropdown.change(raceSetup.loadTrackLayouts);
            raceSetup.$trackLayoutDropdown.change(raceSetup.showTrackImage());
        }

        raceSetup.raceLaps();
        raceSetup.showEnabledSessions();
    },

    /**
     * add weather elements to the form when the 'new weather' button is clicked
     */
    addWeather: function(e) {
        e.preventDefault();

        let $oldWeather = $document.find(".weather").last();

        let $newWeather = $oldWeather.clone(true, true);
        $newWeather.find(".weather-num").text($document.find(".weather").length);

        $oldWeather.after($newWeather);
    },

    /**
     * when a session 'enabled' checkbox is modified, toggle the state of the session-details element
     */
    showEnabledSessions: function() {
        $(".session-enabler").each(function(index, elem) {
            $(elem).on('switchChange.bootstrapSwitch',function(event, state) {
                let $this = $(this);
                let $elem = $this.closest(".tab-pane").find(".session-details");
                let $panelLabel = $document.find("#" + $this.closest(".tab-pane").attr("aria-labelledby"));

                if (state) {
                    $elem.show();
                    $panelLabel.addClass("text-success")
                } else {
                    $elem.hide();
                    $panelLabel.removeClass("text-success")
                }
            });
        });
    },

    // current tyres present in tyres multiselect.
    carTyres: {},

    /**
     * populate the tyre dropdown for all currently selected cars.
     */
    populateTyreDropdown: function() {
        let cars = raceSetup.$carsDropdown.val();

        for (let index = 0; index < cars.length; index++) {
            let car = cars[index];
            let carTyres = availableTyres[car];

            for (let tyre in carTyres) {
                if (raceSetup.carTyres[tyre]) {
                    continue; // this has already been added
                }

                let $opt = $("<option/>");

                $opt.attr({'value': tyre});
                $opt.text(carTyres[tyre] + " (" + tyre + ")");

                raceSetup.$tyresDropdown.append($opt);

                raceSetup.carTyres[tyre] = true;
            }
        }

        raceSetup.$tyresDropdown.multiSelect('refresh');
    },

    /**
     * given a dropdown input which specifies 'laps'/'time', raceLaps will show the correct input element
     * and empty the unneeded one for either laps or race time.
     */
    raceLaps: function() {
        let $timeOrLaps = $document.find("#TimeOrLaps");
        let $raceLaps = $document.find("#RaceLaps");
        let $raceTime = $document.find("#RaceTime");

        if ($timeOrLaps.length) {
            $timeOrLaps.change(function() {
                let selected = $timeOrLaps.find("option:selected").val();

                if (selected === "Time") {
                    $raceLaps.find("input").val(0);
                    $raceTime.find("input").val(15);
                    $raceLaps.hide();
                    $raceTime.show();
                } else {
                    $raceTime.find("input").val(0);
                    $raceLaps.find("input").val(10);
                    $raceLaps.show();
                    $raceTime.hide();
                }
            });
        }
    },

    showTrackImage: function() {
        let track = raceSetup.$trackDropdown.val();
        let layout = raceSetup.$trackLayoutDropdown.val();

        let src = '/content/tracks/' + track + '/ui';

        if (layout) {
            src += '/' + layout;
        }

        src += '/preview.png';

        $document.find("#trackImage").attr({
            'src': src,
            'alt': track + ' ' + layout,
        })
    },

    /**
     * loadTrackLayouts: looks at the selected track and loads in the correct layouts for it into the
     * track layout dropdown
     */
    loadTrackLayouts: function () {
        raceSetup.$trackLayoutDropdown.empty();

        let selectedTrack = raceSetup.$trackDropdown.find("option:selected").val();
        let availableLayouts = raceSetup.trackLayoutOpts[selectedTrack];

        if (availableLayouts) {
            for (let i = 0; i < availableLayouts.length; i++) {
                raceSetup.$trackLayoutDropdown.append(raceSetup.buildTrackLayoutOption(availableLayouts[i]));
            }

            raceSetup.$trackLayoutDropdownParent.show();
        } else {
            // add an option with an empty value
            raceSetup.$trackLayoutDropdown.append(raceSetup.buildTrackLayoutOption(""));
            raceSetup.$trackLayoutDropdownParent.hide();
        }


        raceSetup.showTrackImage();

    },

    /**
     * buildTrackLayoutOption: builds an <option> containing track layout information
     * @param layout
     * @returns {HTMLElement}
     */
    buildTrackLayoutOption: function (layout) {
        let $opt = $("<option/>");
        $opt.attr({'value': layout});
        $opt.text(layout);

        if (layout === raceSetup.currentLayout) {
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

function submitFiles(path) {
    //JSON encode filestoUpload, JQUERY post request to api endpoint (/api/track/car/upload)
    let newFiles = [];
    let count = 0;

    for (let x = 0; x < filesToUpload.length; x++) {
        // Encode and upload, don't post until all files are read
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
                jQuery.post(path, JSON.stringify(newFiles), onSuccess).fail(onFail)
            }
        });
    }
}

function onSuccess(data) {
    console.log("Track Successfully Added");
    location.reload(); // reload for flashes
}

function onFail(data) {
    console.log("Track Could Not be Added");
    location.reload(); // reload for flashes
}

function handleCarFiles(fileList) {
    let filesToUploadLocal = [];

    for (let x = 0; x < fileList.length; x++) {
        // check for multiple cars inside "cars" folder, if so recall this function for each car
        if (fileList[x].webkitRelativePath.startsWith("cars/") && !fileList[x].newPath) {
            let splitList = {};

            for (let y = 0; y < fileList.length; y++) {
                let splitPath = fileList[y].webkitRelativePath.split("/");

                let carIdentifier = splitPath.slice(0, 2).join(":");

                fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

                if (!splitList[carIdentifier]) {
                    splitList[carIdentifier] = []
                }

                splitList[carIdentifier].push(fileList[y]);
            }

            for (let car in splitList) {
                handleCarFiles(splitList[car]);
            }

            return
        }

        // Find the files that the server is interested in
        if (fileList[x].name === "data.acd" || fileList[x].name === "ui_car.json"
            || fileList[x].name.startsWith("livery.") || fileList[x].name.startsWith("preview.")
            || fileList[x].name === "ui_skin.json") {

            filesToUploadLocal.push(fileList[x])
        }
    }

    // Preview panel for the car
    let $carPanel = $("#car-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");
    let previewDone = false;

    let entrySplit = fileList[0].webkitRelativePath.replace('\\', '/').split("/");
    let carName = entrySplit[entrySplit.length-2];

    if (fileList[0].webkitRelativePath.startsWith("cars/")) {
        carName = fileList[0].webkitRelativePath.split("/")[1];
    } else {
        carName = fileList[0].webkitRelativePath.split("/")[0];
    }

    $carPanel.attr({'class': "card p-3 mt-2"});
    $title.text("Preview: " + carName);
    $row.attr({'class': "card-deck"});

    $carPanel.append($title);

    $carPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {

        // Get a preview image, display livery name
        if (filesToUploadLocal[x].name.startsWith("preview.") && !previewDone) {
            previewDone = true;

            let filePathCorrected = filesToUploadLocal[x].webkitRelativePath.replace('\\', '/');
            let filePathSplit = filePathCorrected.split("/");

            let skinName = filePathSplit[filePathSplit.length-2];

            // Set preview to base64 encoded image
            let reader = new FileReader();

            reader.readAsDataURL(filesToUploadLocal[x]);

            reader.addEventListener("load", function () {
                $row.append(buildInfoPanel(reader.result.toString(), "Livery: " + skinName));
            });
        }

        // Get info about the car to display in the preview, this often fails due to bad JSON encoding
        if (filesToUploadLocal[x].name === "ui_car.json") {
            let reader = new FileReader();

            reader.readAsText(filesToUploadLocal[x]);

            reader.addEventListener("load", function () {
                let parsed = "";
                let badJSONnoDonut = false;

                try {
                    parsed = JSON.parse(reader.result.toString());
                }

                catch(error) {
                    badJSONnoDonut = true;
                }

                if (!badJSONnoDonut) {
                    $carPanel.append(buildHtmlTable([parsed]));
                }
            });
        }
    }

    // Create an upload button that sends queued files to the server
    let $uploadButton = $("#upload-button");
    $uploadButton.attr({'class': "d-inline"});

    if (filesToUploadLocal.length === 0) {
        $uploadButton.text("Sorry, the files you uploaded don't seem to be a compatible car!");
        $uploadButton.empty()
    } else {
        if (!$("#car-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({'class': "btn btn-primary", 'onclick': "submitFiles(\"/api/car/upload\")", 'id': "car-upload-button"});
            $button.text("Upload Car(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function handleTrackFiles(fileList) {
    let layouts = {};
    let layoutNum = 0;
    let filesToUploadLocal = [];
    let trackName = "";

    if (fileList[0].webkitRelativePath.startsWith("tracks/")) {
        trackName = fileList[0].webkitRelativePath.split("/")[1];
    } else {
        trackName = fileList[0].webkitRelativePath.split("/")[0];
    }

    for (let x = 0; x < fileList.length; x++) {
        if (fileList[x].webkitRelativePath.startsWith("tracks/") && !fileList[x].newPath) {
            let splitList = {};

            for (let y = 0; y < fileList.length; y++) {
                let splitPath = fileList[y].webkitRelativePath.split("/");

                let trackIdentifier = splitPath.slice(0, 2).join(":");

                fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

                if (!splitList[trackIdentifier]) {
                    splitList[trackIdentifier] = []
                }

                splitList[trackIdentifier].push(fileList[y]);
            }

            for (let track in splitList) {
                handleTrackFiles(splitList[track]);
            }

            return
        }

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
    $title.text("Preview: " + trackName);
    $row.attr({'class': "card-deck"});

    $trackPanel.append($title);

    $trackPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {
        if (filesToUploadLocal[x].webkitRelativePath.includes("/ui/")) {

            if (filesToUploadLocal[x].name === "preview.png") {

                let layoutName = "";

                // For multiple layouts get the layout name and store in map
                if (layoutNum > 1) {
                    let fileListCorrected = filesToUploadLocal[x].webkitRelativePath.replace('\\', '/');

                    let fileListSplit = fileListCorrected.split("/");

                    layoutName = fileListSplit[fileListSplit.length-2];
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

                    $row.append(buildInfoPanel(layoutInfo.preview, "Layout: " + layoutName));
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
        $uploadButton.empty()
    } else {
        if (!$("#track-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({'class': "btn btn-primary", 'onclick': "submitFiles(\"/api/track/upload\")", 'id': "track-upload-button"});
            $button.text("Upload Track(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function buildInfoPanel(img, info) {
    let $panel = $("<div/>");
    let $img = $("<img/>");
    let $cardBody = $("<div/>");
    let $cardText = $("<h5/>");

    $img.attr({'src': img});
    $img.attr({'alt': "Content Preview"});
    $img.attr({'class': "card-img-top"});

    $cardBody.attr({'class': "card-body"});

    $cardText.attr({'class': "card-title"});
    $cardText.text(info);

    $cardBody.append($cardText);

    $panel.append($img);
    $panel.append($cardBody);

    $panel.attr({'class': "card text-center mb-3"});

    return $panel;
}

// Builds a HTML table from JSON input.
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
function addAllColumnHeaders(json, table)
{
    let columnSet = [];
    let headerTr$ = $('<tr/>');
    let header$ = $('<thead/>');

    header$.attr({'class': "table-secondary"});

    for (let i = 0 ; i < json.length ; i++) {
        let rowHash = json[i];
        for (let key in rowHash) {
            if ($.inArray(key, columnSet) === -1){
                if (key === "tags" || key === "run" || key === "url" || key === "torqueCurve" || key === "powerCurve") {
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