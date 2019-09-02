import ChangeEvent = JQuery.ChangeEvent;
import {Connection, jsPlumb} from "jsplumb";
import dagre, {graphlib} from "dagre";
import Graph = graphlib.Graph;

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


        console.log(session1ID, session2ID);
    });

    // construct dagre graph from JsPlumb graph
    const g = new Graph();
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