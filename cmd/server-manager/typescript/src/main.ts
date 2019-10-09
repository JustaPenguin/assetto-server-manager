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
import {RaceWeekend} from "./RaceWeekend";
import {ChangelogPopup} from "./ChangelogPopup";
import {Messages} from "./Messages";
import {Championship} from "./Championship";

$(() => {
    new RaceControl();
    new CarDetail();
    new CarList();
    new RaceWeekend.View();
    new RaceWeekend.EditSession();
    new ChangelogPopup();
    Messages.initSummerNote();
    new Championship.View();

    $(".race-setup").each(function (index, elem) {
        new CarSearch($(elem));
    });
});

declare global {
    interface JQuery {
        multiSelect: any;
    }
}