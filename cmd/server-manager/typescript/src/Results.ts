
export class Results {
    constructor() {
        $(document).on("submit", "#collision-form", Results.processCollisionForm.bind(this));

        $("#show-all-collisions").on("click", Results.showAllCollisions.bind(this));
        $("#hide-all-collisions").on("click", Results.hideAllCollisions.bind(this));

        this.initOpenInSimResults();
        this.initJumpToDriver();
    }

    private initOpenInSimResults(): void {
        $(document).find("#open-in-simres").each(function (index, elem) {
            let link = window.location.href.split("#")[0].replace("results", "results/download") + ".json";

            $(elem).attr('href', "http://simresults.net/remote?result=" + link);

            return false
        });

        $(document).find("#simres-group").each(function (index, elem) {
            let link = window.location.href + "/export-results";

            $(elem).attr('href', "http://simresults.net/remote?results=" + link);

            return false
        });
    }

    private initJumpToDriver(): void {
        $(document).find(".row-link").click(function () {
            window.location = $(this).data("href");
        });

        $(document).find(".results .driver-link").click(function () {
            window.location = $(this).data("href");
            window.scrollBy(0, -100);
        });
    }

    private static processCollisionForm(e: JQuery.SubmitEvent): boolean {
        e.preventDefault();
        e.stopPropagation();

        let formArray = $(e.currentTarget).serializeArray();

        let collisions = "";

        for (let i = 0 ; i < formArray.length ; i++) {
            if (formArray[i].value === "1") {
                collisions += i.toString() + ",";
            }
        }

        let overlayImg = $("#trackImageOverlay");

        overlayImg.attr("src", "/results/" + overlayImg.data("session-file") + "/collisions" + "?collisions=" + collisions);

        let checkboxes = $(".event-checkbox");
        checkboxes.removeAttr("disabled");
        checkboxes.empty();

        return false
    }

    private static showAllCollisions() {
        let checkboxes = $(".event-checkbox");

        checkboxes.bootstrapSwitch("state", true);

        $("#collision-form").submit()
    }

    private static hideAllCollisions() {
        let checkboxes = $(".event-checkbox");

        checkboxes.bootstrapSwitch("state", false);

        $("#collision-form").submit()
    }
}