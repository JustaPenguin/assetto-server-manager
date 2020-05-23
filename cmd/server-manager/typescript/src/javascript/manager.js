"use strict";

let $document;

let moment = require("moment");

// entry-point
$(document).ready(function () {
    console.log("initialising server manager javascript");

    $document = $(document);

    // init bootstrap-switch
    $.fn.bootstrapSwitch.defaults.size = 'small';
    $.fn.bootstrapSwitch.defaults.animate = false;
    $.fn.bootstrapSwitch.defaults.onColor = "success";
    $document.find("input[type='checkbox']:not(input[name='EntryList.OverwriteAllEvents']:hidden):not(input[name='session-start-after-parent']:hidden)").bootstrapSwitch();

    serverLogs.init();
    initUploaders();

    $document.find('[data-toggle="tooltip"]').tooltip();

    $("[data-toggle=popover]").each(function (i, obj) {
        $(this).popover({
            html: true,
            sanitize: false,
            content: function () {
                let id = $(this).attr('id');

                return $('#popover-content-' + id).html();
            },
        });
    });

    $(".time-local").each(function (i, elem) {
        let $elem = $(elem);

        $elem.text(moment.parseZone($elem.attr("data-time")).tz(moment.tz.guess()).format('LLLL (z)'));
    });

    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

    $(".timezone").text(timezone);
    $(".event-schedule-timezone").val(timezone);
    $(".session-schedule-timezone").val(timezone);
    $(".sol-timezone").val(timezone);


    $('form').submit(function () {
        $(this).find('input[type="checkbox"]').each(function () {
            let $checkbox = $(this);
            if ($checkbox.is(':checked')) {
                $checkbox.attr('value', '1');
            } else {
                $checkbox.after().append($checkbox.clone().attr({type: 'hidden', value: 0}));
                $checkbox.prop('disabled', true);
            }
        })
    });

    if ($document.find("form[data-safe-submit]").length > 0) {
        let canSubmit = false;

        $document.find("button[type='submit']").click(function () {
            canSubmit = true;
        });

        // ask the user before they close the webpage
        window.onbeforeunload = function () {
            if (canSubmit) {
                return;
            }

            return "Are you sure you want to navigate away? You'll lose unsaved changes to this setup if you do.";
        };
    }

    // Fix for mobile safari
    $document.find(".Cars").change(function () {
        let selectedCars = $(this).children("option:selected").val();

        if (selectedCars !== undefined) {
            $(this).removeAttr('required')
        } else {
            $(this).attr('required', 'required')
        }
    });

    $('#SetupFile').on('change', function () {
        let fileName = this.files[0].name;
        $(this).next('.custom-file-label').html(fileName);
    });

    $document.find("#CustomRaceScheduled").change(function () {
        if ($(this).val() && $document.find("#CustomRaceScheduledTime").val()) {
            $document.find("#start-race-button").hide();
            $document.find("#save-race-button").val("schedule");
        } else {
            $document.find("#start-race-button").show();
            $document.find("#save-race-button").val("justSave");
        }
    });

    $document.find("#CustomRaceScheduledTime").change(function () {
        if ($(this).val() && $document.find("#CustomRaceScheduled").val()) {
            $document.find("#start-race-button").hide();
            $document.find("#save-race-button").val("schedule");
        } else {
            $document.find("#start-race-button").show();
            $document.find("#save-race-button").val("justSave");
        }
    });
});

export function initMultiSelect($element) {
    $element.each(function (i, elem) {
        let $elem = $(elem);

        if ($elem.is(":hidden")) {
            return true;
        }

        $elem.multiSelect();
    });
}

let serverLogs = {
    init: function () {
        let $serverLog = $document.find("#server-logs");
        let $managerLog = $document.find("#manager-logs");
        let $pluginLog = $document.find("#plugin-logs");

        let disableServerLogRefresh = false;
        let disableManagerLogRefresh = false;
        let disablePluginLogRefresh = false;

        $serverLog.on("mousedown", function () {
            disableServerLogRefresh = true;
        });

        $serverLog.on("mouseup", function () {
            disableServerLogRefresh = false;
        });

        $pluginLog.on("mousedown", function () {
            disablePluginLogRefresh = true;
        });

        $pluginLog.on("mouseup", function () {
            disablePluginLogRefresh = false;
        });

        $managerLog.on("mousedown", function () {
            disableManagerLogRefresh = true;
        });

        $managerLog.on("mouseup", function () {
            disableManagerLogRefresh = false;
        });

        function isAtBottom($elem) {
            let node = $elem[0];
            return node.scrollTop + node.offsetHeight >= node.scrollHeight - 40;
        }

        if ($serverLog.length && $managerLog.length && $pluginLog.length) {
            setInterval(function () {
                $.get("/api/logs", function (data) {
                    if (!window.getSelection().toString()) {

                        if (isAtBottom($serverLog) && !disableServerLogRefresh) {
                            $serverLog.text(data.ServerLog);
                            $serverLog.scrollTop(1E10);
                        }

                        if (isAtBottom($managerLog) && !disableManagerLogRefresh) {
                            $managerLog.text(data.ManagerLog);
                            $managerLog.scrollTop(1E10);
                        }

                        if (isAtBottom($pluginLog) && !disablePluginLogRefresh) {
                            $pluginLog.text(data.PluginsLog);
                            $pluginLog.scrollTop(1E10);
                        }
                    }
                });
            }, 1000);
        }
    },
};


function initUploaders() {
    $("#input-folder-car").on("change", function () {
        handleCarFiles(this.files);
    });

    $("#drop-zone.car-drop").on("drop", function (e) {
        handleCarDropFiles(e);
    });

    $("#drop-zone").on("dragover", dragOverHandler);
    $("#drop-zone").on("dragleave", dragOutHandler);
    $("#only-ks").on("switchChange.bootstrapSwitch", toggleKS);

    $("#input-folder-track").on("change", function () {
        handleTrackFiles(this.files);
    });

    $("#drop-zone.track-drop").on("drop", function (e) {
        handleTrackDropFiles(e);
    });

    $("#input-folder-weather").on("change", function () {
        handleWeatherFiles(this.files);
    });

    $("#drop-zone.weather-drop").on("drop", function (e) {
        handleWeatherDropFiles(e);
    });
}


function postWithProgressBar(path, data, onSuccess, onFail, $progressBar) {
    $progressBar.closest(".progress").show();
    $progressBar.removeClass("bg-success");

    function showProgress(evt) {
        if (evt.lengthComputable) {
            let percentComplete = Math.round((evt.loaded / evt.total) * 100);
            $progressBar.css('width', percentComplete + '%').attr('aria-valuenow', percentComplete);
            $progressBar.text(percentComplete + "%");

            if (percentComplete === 100) {
                $progressBar.addClass("bg-success");
            }
        }
    }

    $.ajax({
        xhr: function () {
            let xhr = new window.XMLHttpRequest();
            xhr.upload.addEventListener("progress", showProgress, false);
            xhr.addEventListener("progress", showProgress, false);
            return xhr;
        },
        type: 'POST',
        url: path,
        data: data,
        success: onSuccess,
        fail: onFail,
    });
}

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
                'name': filesToUpload[x].name,
                'size': filesToUpload[x].size,
                'type': filesToUpload[x].type,
                'filepath': filesToUpload[x].filepath,
                'dataBase64': reader.result.toString()
            });

            count++;

            if (count === filesToUpload.length) {
                postWithProgressBar(path, JSON.stringify(newFiles), onSuccess, onFail, $("#progress-bar"));
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

function handleWeatherFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    let idPos = 0;

    if (fileList[0].filepath.startsWith("weather/")) {
        idPos = 2
    } else {
        idPos = 1
    }

    let splitList = {};

    for (let y = 0; y < fileList.length; y++) {
        let splitPath = fileList[y].filepath.split("/");

        let weatherIdentifier = splitPath.slice(0, idPos).join(":");

        fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

        if (!splitList[weatherIdentifier]) {
            splitList[weatherIdentifier] = []
        }

        splitList[weatherIdentifier].push(fileList[y]);
    }

    for (let weather in splitList) {
        handleWeatherFilesLoop(splitList[weather]);
    }
}

function handleWeatherFilesLoop(fileList) {
    let filesToUploadLocal = [];
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // Find the files that the server is interested in
        if (fileList[x].name === "weather.ini" || fileList[x].name.startsWith("preview.")) {
            filesToUploadLocal.push(fileList[x]);

            goodFile = true;
        }
    }

    if (!goodFile) {
        return
    }

    // Preview panel for the weather preset
    let $weatherPanel = $("#weather-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");
    let previewDone = false;

    let weatherName = "";

    if (fileList[0].filepath.startsWith("weather/")) {
        weatherName = fileList[0].filepath.replace('\\', '/').split("/")[1];
    } else {
        weatherName = fileList[0].filepath.replace('\\', '/').split("/")[0];
    }

    $weatherPanel.attr({'class': "card p-3 mt-2"});
    $title.text("Preview: " + weatherName);
    $row.attr({'class': "card-deck"});

    $weatherPanel.append($title);

    $weatherPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {

        // Get a preview image, display livery name
        if (filesToUploadLocal[x].name.startsWith("preview.") && !previewDone) {
            previewDone = true;

            // Set preview to base64 encoded image
            let reader = new FileReader();

            reader.readAsDataURL(filesToUploadLocal[x]);

            reader.addEventListener("load", function () {
                $row.append(buildInfoPanel(reader.result.toString(), "Weather Preview"));
            });
        }
    }

    // Create an upload button that sends queued files to the server
    let $uploadButton = $("#upload-button");
    $uploadButton.attr({'class': "d-inline"});

    if (filesToUploadLocal.length === 0) {
        $uploadButton.text("Sorry, the files you uploaded don't seem to be a compatible weather preset!");
        $uploadButton.empty()
    } else {
        if (!$("#weather-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({
                'class': "btn btn-primary",
                'id': "weather-upload-button"
            });

            $button.click(function (e) {
                e.preventDefault();
                submitFiles("/api/weather/upload")
            });
            $button.text("Upload Weather Preset(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

let onlyKS = false;

function toggleKS() {
    onlyKS = $document.find("#only-ks").is(':checked');
}

function handleCarFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    let idPos = 0;

    if (fileList[0].filepath.startsWith("cars/")) {
        idPos = 2
    } else {
        idPos = 1
    }

    let splitList = {};

    for (let y = 0; y < fileList.length; y++) {
        // if onlyKS is set, and the file doesn't contain the ks_ prefix
        if (onlyKS && fileList[y].filepath.indexOf("ks_", 0) === -1) {
            continue
        }

        let splitPath = fileList[y].filepath.split("/");

        let carIdentifier = splitPath.slice(0, idPos).join(":");

        fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

        if (!splitList[carIdentifier]) {
            splitList[carIdentifier] = []
        }

        splitList[carIdentifier].push(fileList[y]);
    }

    for (let car in splitList) {
        handleCarFilesLoop(splitList[car]);
    }
}

function handleCarFilesLoop(fileList) {
    let filesToUploadLocal = [];
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // Find the files that the server is interested in
        if (fileList[x].name === "data.acd" || fileList[x].name === "tyres.ini" || fileList[x].name === "ui_car.json"
            || fileList[x].name.startsWith("livery.") || fileList[x].name.startsWith("preview.")
            || fileList[x].name === "ui_skin.json" || fileList[x].filepath.indexOf("/data/") !== -1) {

            filesToUploadLocal.push(fileList[x]);

            if (fileList[x].name === "ui_car.json") {
                goodFile = true;
            }
        }
    }

    if (!goodFile) {
        notA("car");
        return
    }

    let $panel = $("#car-fail");
    $panel.hide();

    // Preview panel for the car
    let $carPanel = $("#car-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");
    let previewDone = false;

    let entrySplit = fileList[0].filepath.replace('\\', '/').split("/");
    let carName = entrySplit[entrySplit.length - 2];

    if (fileList[0].filepath.startsWith("cars/")) {
        carName = fileList[0].filepath.split("/")[1];
    } else {
        carName = fileList[0].filepath.split("/")[0];
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

            let filePathCorrected = filesToUploadLocal[x].filepath.replace('\\', '/');
            let filePathSplit = filePathCorrected.split("/");

            let skinName = filePathSplit[filePathSplit.length - 2];

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
                } catch (error) {
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
            $button.attr({
                'class': "btn btn-primary",
                'id': "car-upload-button"
            });

            $button.click(function (e) {
                e.preventDefault();

                submitFiles("/api/car/upload")
            });
            $button.text("Upload Car(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function getFilesWebkitDataTransferItems(dataTransferItems) {
    function traverseFileTreePromise(item, path = '') {
        return new Promise(resolve => {
            if (item.isFile) {
                item.file(file => {
                    file.filepath = path + file.name; //save full path
                    files.push(file);
                    resolve(file)
                })
            } else if (item.isDirectory) {
                let dirReader = item.createReader();
                dirReader.readEntries(entries => {
                    let entriesPromises = [];

                    for (let entr of entries) {
                        entriesPromises.push(traverseFileTreePromise(entr, path + item.name + "/"));
                    }

                    resolve(Promise.all(entriesPromises))
                })
            }
        })
    }

    let files = [];
    return new Promise((resolve, reject) => {
        let entriesPromises = [];

        for (let it of dataTransferItems) {
            entriesPromises.push(traverseFileTreePromise(it.webkitGetAsEntry()));
        }

        Promise.all(entriesPromises)
            .then(entries => {
                resolve(files)
            })
    })
}

function handleTrackDropFiles(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    dragOutHandler(ev);

    let items = event.dataTransfer.items;
    getFilesWebkitDataTransferItems(items)
        .then(files => {
            handleTrackFiles(files);
        })
}

function handleCarDropFiles(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    dragOutHandler(ev);

    let items = event.dataTransfer.items;
    getFilesWebkitDataTransferItems(items)
        .then(files => {
            handleCarFiles(files);
        })
}

function handleWeatherDropFiles(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    dragOutHandler(ev);

    let items = event.dataTransfer.items;
    getFilesWebkitDataTransferItems(items)
        .then(files => {
            handleWeatherFiles(files);
        })
}

function dragOverHandler(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    document.getElementById("drop-zone").classList.add('drop-zone-hovered');
}

function dragOutHandler(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    document.getElementById("drop-zone").classList.remove('drop-zone-hovered');
}

function handleTrackFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    let idPos = 0;

    if (fileList[0].filepath.startsWith("tracks/")) {
        idPos = 2
    } else {
        idPos = 1
    }

    let splitList = {};

    for (let y = 0; y < fileList.length; y++) {
        // if onlyKS is set, and the file doesn't contain the ks_ prefix
        if (onlyKS && fileList[y].filepath.indexOf("ks_", 0) === -1) {
            continue
        }

        let splitPath = fileList[y].filepath.split("/");

        let trackIdentifier = splitPath.slice(0, idPos).join(":");

        fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

        if (!splitList[trackIdentifier]) {
            splitList[trackIdentifier] = []
        }

        splitList[trackIdentifier].push(fileList[y]);
    }

    for (let track in splitList) {
        handleTrackFilesLoop(splitList[track]);
    }

}

function handleTrackFilesLoop(fileList) {
    let layouts = {};
    let layoutNum = 0;
    let filesToUploadLocal = [];
    let trackName = "";
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // get model/surfaces and drs zones and ui folder
        if ((fileList[x].name.startsWith("models") && fileList[x].name.endsWith(".ini")) ||
            (fileList[x].name === "surfaces.ini" || fileList[x].name === "drs_zones.ini") ||
            (fileList[x].filepath.includes("/ui/") || fileList[x].name === "map.png" || fileList[x].name === "map.ini")) {

            filesToUploadLocal.push(fileList[x]);
        }

        if (fileList[x].name.startsWith("models")) {
            layoutNum++
        }

        if (fileList[x].name === "surfaces.ini") {
            goodFile = true;
        }
    }

    if (!goodFile) {
        notA("track");
        return
    }

    let $panel = $("#track-fail");
    $panel.hide();

    if (fileList[0].filepath.startsWith("tracks/")) {
        trackName = fileList[0].filepath.split("/")[1];
    } else {
        trackName = fileList[0].filepath.split("/")[0];
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
        if (filesToUploadLocal[x].filepath.includes("/ui/")) {

            if (filesToUploadLocal[x].name === "preview.png") {

                let layoutName = "";

                // For multiple layouts get the layout name and store in map
                if (layoutNum > 1) {
                    let fileListCorrected = filesToUploadLocal[x].filepath.replace('\\', '/');

                    let fileListSplit = fileListCorrected.split("/");

                    layoutName = fileListSplit[fileListSplit.length - 2];
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
            $button.attr({
                'class': "btn btn-primary",
                'id': "track-upload-button"
            });
            $button.click(function (e) {
                e.preventDefault();

                submitFiles("/api/track/upload")
            });

            $button.text("Upload Track(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function notA(thing) {
    let $panel = $("#" + thing + "-fail");

    $panel.show();
    $panel.attr({'class': "alert alert-danger mt-2"});
    $panel.text("Sorry, looks like that wasn't a " + thing + "!")
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

    for (let i = 0; i < json.length; i++) {
        let $row = $('<tr/>');
        for (let colIndex = 0; colIndex < columns.length; colIndex++) {
            let cellValue = json[i][columns[colIndex]] + "<br>";

            if (cellValue == null) {
                cellValue = "";
            }

            $row.append($('<td/>').html(cellValue));
        }
        $cardTable.append($row);
    }

    return $cardTable
}

// Adds a header row to the table and returns the set of columns.
function addAllColumnHeaders(json, table) {
    let columnSet = [];
    let headerTr$ = $('<tr/>');
    let header$ = $('<thead/>');

    header$.attr({'class': "table-secondary"});

    for (let i = 0; i < json.length; i++) {
        let rowHash = json[i];
        for (let key in rowHash) {
            if ($.inArray(key, columnSet) === -1) {
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

