import "jquery";
import "bootstrap";
import "bootstrap-switch";
import "summernote/dist/summernote-bs4";
import "multiselect";
import "moment";
import "moment-timezone";
import "./javascript/manager.js";
import "./Font";
import "./Calendar";
import "./RaceList";

import {RaceControl} from "./RaceControl";
import {CarDetail} from "./CarDetail";
import {CarSearch} from "./CarSearch";
import {CarList} from "./CarList";
import {RaceWeekendSession} from "./RaceWeekend";
import {ChangelogPopup} from "./ChangelogPopup";

$(() => {
    new RaceControl();
    new CarDetail();
    new CarList();
    new RaceWeekendSession();
    new ChangelogPopup();

    $(".race-setup").each(function (index, elem) {
        new CarSearch($(elem));
    });
});

declare global {
    interface JQuery {
        multiSelect: any;
    }
}