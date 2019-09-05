import ChangeEvent = JQuery.ChangeEvent;
import {Connection, jsPlumb} from "jsplumb";
import dagre, {graphlib} from "dagre";

declare var RaceWeekendID: string;

export class RaceWeekendSession {
    private $raceWeekendSession: JQuery<HTMLElement>;

    public constructor() {
        this.$raceWeekendSession = $("#race-weekend-session");

        if (!this.$raceWeekendSession.length) {
            return;
        }

        this.initSessionTypeSwitch();
    }

    private initSessionTypeSwitch(): void {
        const $sessionSwitcher = this.$raceWeekendSession.find("#SessionType");

        $sessionSwitcher.on("change", (e: ChangeEvent) => {
            const val = $(e.currentTarget).val();
            this.$raceWeekendSession.find(".sessions .tab-pane").removeClass(["show", "active"]);

            const $newSession = this.$raceWeekendSession.find("#session-" + val);
            $newSession.addClass(["show", "active"]);
            $newSession.find(".session-details").show();

            this.$raceWeekendSession.find(".session-enabler").prop("checked", false);
            this.$raceWeekendSession.find("#" + val + "\\.Enabled").prop("checked", true);
        });
    }
}

const jsp = jsPlumb.getInstance();

jsp.bind("ready", () => {
    jsp.importDefaults({
        ConnectionsDetachable: false,
        ReattachConnections: false,
    });

    $(".race-weekend-session").each((index, element) => {
        const $session = $(element);

        // @ts-ignore
        jsp.draggable(element, {grid: [10, 10]});

        const parentIDs = JSON.parse($session.data("parent-ids")) as string[];

        for (let parentID of parentIDs) {
            let conn = jsp.connect({
                source: parentID,
                target: $session.attr("id"),
                anchor: "AutoDefault",
                // @ts-ignore
                endpoint: ["Blank", {width: 10, height: 10}],

                connector: ["Flowchart", {}],

            });

            if (conn) {
                // @ts-ignore
                conn.addOverlay(["PlainArrow", {width: 20, height: 20, id: "arrow"}]);
            }
        }
    });

    jsp.bind("click", (conn: Connection, originalEvent: Event): void => {
        const [ep1, ep2] = conn.endpoints;

        let session1ID = $(ep1.getElement()).attr("id");
        let session2ID = $(ep2.getElement()).attr("id");


        const modalContentURL = `/race-weekend/${RaceWeekendID}/filters?parentSessionID=${session1ID}&childSessionID=${session2ID}`;

        $.get(modalContentURL).then((data: string) => {
            let $filtersModal = $("#filters-modal");
            $filtersModal.html(data);
            $filtersModal.find("input[type='checkbox']").bootstrapSwitch();
            $filtersModal.modal();

            new RaceWeekendSessionTransition($filtersModal, session1ID!, session2ID!);
        });
    });

    // construct dagre graph from JsPlumb graph
    const g = new graphlib.Graph();
    g.setGraph({
        nodesep: 550,
    });
    g.setDefaultEdgeLabel(function () {
        return {};
    });

    $('.race-weekend-session').each((idx, node) => {
        const n = $(node);
        g.setNode(n.attr('id')!, {
            width: Math.round(n.width()!),
            height: Math.round(n.height()!)
        });
    });

    for (const edge of jsp.getAllConnections() as any[]) {
        g.setEdge(
            edge.source.id,
            edge.target.id
        );
    }

    // calculate node positions
    dagre.layout(g);

    $("#race-weekend-graph-container").css({
        "height": (g.graph().height! + 200) + "px",
        "width": (g.graph().width! + 350) + "px",
    });

    // apply node positions
    g.nodes().forEach((n) => {
        let $n = $('#' + n);
        $n.css('left', g.node(n).x + 'px');
        $n.css('top', g.node(n).y + 'px');
    });

    jsp.repaintEverything();
});

export class RaceWeekendSessionTransition {
    private $elem: JQuery<HTMLElement>;
    private parentSessionID: string;
    private childSessionID: string;

    private resultStart!: number;
    private resultEnd!: number;
    private reverseOrder!: boolean;
    private gridStart!: number;
    private gridEnd!: number;

    public constructor($elem: JQuery<HTMLElement>, parentSessionID: string, childSessionID: string) {
        this.$elem = $elem;
        this.parentSessionID = parentSessionID;
        this.childSessionID = childSessionID;

        this.updateValues();
        this.registerEvents();
    }

    private registerEvents(): void {
        this.$elem.find("input").on("change", () => {
            this.updateValues();
        });

        this.$elem.find("input").on("switchChange.bootstrapSwitch", () => {
            this.updateValues();
        });

        this.$elem.find("#save-filters").on("click", () => {
            this.saveValues();
        });
    }

    private packageValues(): string {
        return JSON.stringify({
            ResultStart: this.resultStart,
            ResultEnd: this.resultEnd,
            ReverseEntrants: this.reverseOrder,
            EntryListStart: this.gridStart,
            EntryListEnd: this.gridEnd,
        })
    }

    private updateValues(): void {
        this.resultStart = parseInt(this.$elem.find("#ResultsStart").val() as string);
        this.resultEnd = parseInt(this.$elem.find("#ResultsEnd").val() as string);
        this.reverseOrder = this.$elem.find("#ReverseGrid").is(":checked");
        this.gridStart = parseInt(this.$elem.find("#GridStart").val() as string);
        this.gridEnd = parseInt(this.$elem.find("#GridEnd").val() as string);

        $.ajax(`/race-weekend/${RaceWeekendID}/grid-preview?parentSessionID=${this.parentSessionID}&childSessionID=${this.childSessionID}`, {
            data: this.packageValues(),

            contentType: "application/json",
            type: "POST",
        }).then((response: GridPreview) => {
            console.log(response);

            let $resultsPreview = $("#results-preview");
            let $gridPreview = $("#grid-preview");

            $resultsPreview.empty();
            $gridPreview.empty();

            for (const [key, value] of Object.entries(response.Results)) {
                $resultsPreview.append(`${key}. ${value}<br>`);
            }

            for (const [key, value] of Object.entries(response.Grid)) {
                $gridPreview.append(`${key}. ${value}<br>`);
            }
        });
    }

    private saveValues(): void {
        $.ajax(`/race-weekend/${RaceWeekendID}/grid?parentSessionID=${this.parentSessionID}&childSessionID=${this.childSessionID}`, {
            data: this.packageValues(),

            contentType: "application/json",
            type: "POST",
        }).then(() => {
            $("#filters-modal").modal("hide");
        }); // @TODO error handling
    }
}

interface GridPreview {
    Grid: Map<number, string>;
    Results: Map<number, string>;
}