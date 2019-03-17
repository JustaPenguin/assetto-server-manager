import $ from "jquery";
import {ordinalSuffix} from "./Utils";
import {RaceSetup} from "./RaceSetup";

declare var defaultPoints: any;

export class Championship {
    private $document: JQuery<Document>;
    private $pointsTemplate: JQuery<HTMLElement>;
    private $classTemplate: JQuery<HTMLElement>;

    constructor() {
        this.$document = $(document);
        this.$pointsTemplate = $(".points-place").last().clone();

        let $tmpl = $("#class-template");
        this.$classTemplate = $tmpl.clone();
        $tmpl.remove();

        this.$document.on("click", ".addEntrant", (e: JQuery.ClickEvent) => {
            this.onAddEntrantClick(e)
        });

        this.$document.find("#addClass").on("click", (e: JQuery.ClickEvent) => {
            this.onAddClassButtonClick(e);
        });

        this.$document.on("click", ".btn-delete-class", (e: JQuery.ClickEvent) => {
            e.preventDefault();
            $(e.currentTarget).closest(".race-setup").remove();
        });
    }

    private onAddEntrantClick(e: JQuery.ClickEvent): void {
        e.preventDefault();

        let $raceSetup = $(e.currentTarget).closest(".race-setup");
        let $pointsParent = $raceSetup.find(".points-parent");

        if (!$pointsParent.length) {
            return;
        }

        let $points = $raceSetup.find(".points-place");
        let numEntrants = $raceSetup.find(".entrant:visible").length;
        let numPoints = $points.length;

        for (let i = numPoints; i < numEntrants; i++) {
            // add points up to the numEntrants we have
            let $newPoints = this.$pointsTemplate.clone();
            $newPoints.find("label").text(ordinalSuffix(i + 1) + " Place");

            let pointsVal = 0;

            // load the default points value for this position
            if (i < defaultPoints.Places.length) {
                pointsVal = defaultPoints.Places[i];
            }

            $newPoints.find("input").attr({"value": pointsVal});
            $pointsParent.append($newPoints);
        }
    }

    private onAddClassButtonClick(e: JQuery.ClickEvent): void {
        e.preventDefault();

        let $cloned = this.$classTemplate.clone().show();

        $(e.currentTarget).before($cloned);
        new RaceSetup($cloned);
    }
}