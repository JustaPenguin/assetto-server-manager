import "jquery";
import "bootstrap";
import "bootstrap-switch";
import "summernote/dist/summernote-bs4";
import "multiselect";
import "moment";
import "moment-timezone";
import "./javascript/manager.js";
import {RaceControl} from "./RaceControl";
import "./Font";
import {CarDetail} from "./CarDetail";
import "./Calendar";
import {CarSearch} from "./CarSearch";
import {CarList} from "./CarList";
import "./RaceList";

$(() => {
    new RaceControl();
    new CarDetail();
    new CarList();

    $(".race-setup").each(function (index, elem) {
        new CarSearch($(elem));
    });
});

declare global {
    interface JQuery {
        multiSelect: any;
    }
}