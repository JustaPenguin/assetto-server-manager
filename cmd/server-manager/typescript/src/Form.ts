import select2 from "select2";

export class Form {
    public constructor() {
        this.initSelect2();
    }

    private initSelect2() {
        // initialise select2 using its own batshit loader.
        select2($);
    }

    public static initialiseSelect2InElement($elem: JQuery<Document | HTMLElement>) {
        Form.initialiseSelect2OnElement($elem.find("select:not([multiple])"));
    }

    public static initialiseSelect2OnElement($elem) {
        $elem.select2({
            theme: "bootstrap4",
            templateResult: (data) => {
                let $elem = $(data.element);

                let trackName = $elem.data("track-name");

                if (!trackName) {
                    return data.text;
                }

                let $opt = $("<span>");
                $opt.text($elem.text());
                $opt.append($("<small class='float-right text-muted'>" + trackName + "</small>"));
                $opt.append($("<div class='clearfix'></div>"));

                return $opt;
            },
        });
    }
}
