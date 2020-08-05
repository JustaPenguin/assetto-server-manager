import ChangeEvent = JQuery.ChangeEvent;
import ClickEvent = JQuery.ClickEvent;
import {Connection, jsPlumb, jsPlumbInstance} from "jsplumb";
import dagre, {graphlib} from "dagre";
import {initMultiSelect} from "./javascript/manager";

declare var RaceWeekendID: string;
declare var IsEditing: boolean;

export namespace RaceWeekend {
    /**
     * EditSession manages page functions when editing the race configuration for a Race Weekend Session.
     */
    export class EditSession {
        private readonly $raceWeekendSession: JQuery<HTMLElement>;

        public constructor() {
            this.$raceWeekendSession = $("#race-weekend-session");

            if (!this.$raceWeekendSession.length) {
                return;
            }

            this.initSessionTypeSwitch();

            initMultiSelect($("#ParentSessions"));
        }

        private initSessionTypeSwitch(): void {
            const $sessionSwitcher = this.$raceWeekendSession.find("#SessionType");

            if (!IsEditing) {
                this.handleSessionPoints($sessionSwitcher.val() as string);
            }

            $sessionSwitcher.on("change", (e: ChangeEvent) => {
                const val = $(e.currentTarget).val();
                this.$raceWeekendSession.find(".sessions .tab-pane").removeClass(["show", "active"]);

                const $newSession = this.$raceWeekendSession.find("#session-" + val);
                $newSession.addClass(["show", "active"]);
                $newSession.find(".session-details").show();

                this.$raceWeekendSession.find(".session-enabler").prop("checked", false);
                this.$raceWeekendSession.find("#" + val + "\\.Enabled").prop("checked", true);

                this.handleSessionPoints(val as string);
            });
        }

        private handleSessionPoints(sessionType: string) {
            if (sessionType !== "Race") {
                $(".init-empty-non-race").val(0);
            } else {
                $(".init-empty-non-race").each((index: number, elem: HTMLElement) => {
                    let $elem = $(elem);

                    $elem.val($elem.data("default-value"));
                });

                // set wait time for races to be a large number
                $("#Race\\.WaitTime").val(300);
            }
        }
    }

    /**
     * View handles layout of the Race Weekend main page, as well as initialising Entry List popups.
     */
    export class View {
        private readonly jsp: jsPlumbInstance = jsPlumb.getInstance();

        public constructor() {
            this.jsp.bind("ready", () => {
                this.initJsPlumb();
            });

            $(".view-results").on("click", this.onViewResultsClick);
            $(".manage-entrylist").on("click", this.openManageEntryListModal);

            this.initSessionDetailsButtons();
        }

        private initJsPlumb(): void {
            this.jsp.importDefaults({
                ConnectionsDetachable: false,
                ReattachConnections: false,
            });

            $(".race-weekend-session").each((index, element) => {
                const $session = $(element);

                // @ts-ignore
                this.jsp.draggable(element, {grid: [5, 5]});

                const parentIDsJSON = $session.data("parent-ids");

                if (parentIDsJSON) {
                    const parentIDs = JSON.parse(parentIDsJSON) as string[];

                    for (let parentID of parentIDs) {
                        let conn = this.jsp.connect({
                            source: parentID,
                            target: $session.attr("id"),
                            anchor: "AutoDefault",
                            // @ts-ignore
                            endpoint: ["Blank", {width: 10, height: 10}],

                            connector: ["Flowchart", {cornerRadius: 10}],
                            cssClass: "race-weekend-connector",
                        });

                        if (conn) {
                            // @ts-ignore
                            conn.addOverlay(["PlainArrow", {
                                width: 20,
                                height: 20,
                                id: "arrow",
                                cssClass: "race-weekend-arrow"
                            }]);
                        }
                    }
                }
            });

            this.jsp.bind("click", (conn: Connection, originalEvent: Event): void => {
                const [ep1, ep2] = conn.endpoints;

                let session1ID = $(ep1.getElement()).attr("id");
                let session2ID = $(ep2.getElement()).attr("id");

                this.openManageFilterModal(session1ID!, session2ID!);
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

            for (const edge of this.jsp.getAllConnections() as any[]) {
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

            this.jsp.repaintEverything();
        }

        private openManageFilterModal(session1ID: string, session2ID: string): void {
            const modalContentURL = `/race-weekend/${RaceWeekendID}/filters?parentSessionID=${session1ID}&childSessionID=${session2ID}`;

            $.get(modalContentURL).then((data: string) => {
                let $filtersModal = $("#filters-modal");
                $filtersModal.html(data);
                $filtersModal.find("input[type='checkbox']").bootstrapSwitch();
                $filtersModal.modal();

                new SessionTransition($filtersModal, session1ID, session2ID);
            });
        }

        private openManageEntryListModal(e: JQuery.ClickEvent): void {
            e.preventDefault();

            const sessionID = $(e.currentTarget).closest(".race-weekend-session").attr("id") as string;

            const modalContentURL = `/race-weekend/${RaceWeekendID}/entrylist?sessionID=${sessionID}`;

            $.get(modalContentURL).then((data: string) => {
                let $filtersModal = $("#filters-modal");
                $filtersModal.html(data);
                $filtersModal.find("input[type='checkbox']").bootstrapSwitch();
                $filtersModal.modal();

                new EntryListPreview($filtersModal, sessionID);
            });
        }

        private onViewResultsClick(e: JQuery.ClickEvent) {
            e.preventDefault();

            let $raceWeekendSession = $(this).closest(".race-weekend-session");
            let sessionID = $raceWeekendSession.attr("id");

            let $results = $("#results-" + sessionID);

            $('html, body').animate({
                scrollTop: ($("#race-weekend-results").offset()!.top) - 200,
            }, 500, () => {
                $results.collapse('show');
            });
        }

        private initSessionDetailsButtons(): void {
            $(document).on("click", ".race-weekend-session-details", (e: ClickEvent) => {
                let $this = $(e.currentTarget);
                let sessionID = $this.attr("data-session-id");

                const modalContentURL = `/event-details?raceWeekendID=${RaceWeekendID}&sessionID=${sessionID}`;

                $.get(modalContentURL).then((data: string) => {
                    let $eventDetailsModal = $("#session-details-modal");
                    $eventDetailsModal.html(data);
                    $eventDetailsModal.find("input[type='checkbox']").bootstrapSwitch();
                    $eventDetailsModal.modal();
                });

                return false;
            });
        }
    }

    /**
     * PreviewModal is an abstract base for building Race Weekend Entry List preview modals
     */
    abstract class PreviewModal {
        protected readonly $elem: JQuery<HTMLElement>;

        protected constructor($elem: JQuery<HTMLElement>) {
            this.$elem = $elem;
        }

        protected registerEvents(): void {
            this.$elem.find("input, select").on("change", () => {
                this.updateValues();
            });

            this.$elem.find("input").on("switchChange.bootstrapSwitch", () => {
                this.updateValues();
            });

            this.$elem.find("#save-filters").on("click", () => {
                this.saveValues();
            });
        }

        protected abstract updateValues(): void;

        protected abstract saveValues(): void;

        protected buildTableDataForEntrant(entrant: SessionPreviewEntrant, pos?: number): JQuery<HTMLTableDataCellElement> {
            let $td = $("<td>") as JQuery<HTMLTableDataCellElement>;

            if (entrant.Class) {
                $td.css({"background-color": entrant.ClassColor, "color": "white"});
            }

            if (pos !== undefined) {
                $td.text(`${pos + 1}. ${entrant.Name}`);
            } else {
                $td.text(entrant.Name);
            }

            return $td;
        }

        protected buildClassKey(classes: Map<string, string>) {
            let $tableKey = $("#table-key");
            $tableKey.empty();

            if (Object.entries(classes).length > 1) {
                for (const [className, classColor] of Object.entries(classes)) {
                    let $colorBlock = $("<div>").attr({
                        "class": "class-key__background",
                    }).css("background-color", classColor);
                    let $colorText = $("<div>").attr({"class": "class-key__name"}).text(className);

                    $tableKey.append($("<div>").attr({"class": "class-key"}).append($colorBlock, $colorText));
                }
            }
        }
    }

    enum SplitType {
        Numeric = "Numeric",
        ManualDriverSelection = "Manual Driver Selection",
        ChampionshipClass = "Championship Class",
    }

    /**
     * SessionTransition is the modal shown when a user clicks on an arrow between two Race Weekend Sessions.
     */
    class SessionTransition extends PreviewModal {
        private readonly parentSessionID: string;
        private readonly childSessionID: string;

        private resultStart!: number;
        private resultEnd!: number;
        private reverseGrid: number = 0;
        private gridStart!: number;
        private sortType!: string;
        private availableResultsForSorting: string[] = [];
        private startOnFastestLapTyre: boolean = false;
        private splitType: SplitType = SplitType.Numeric;
        private selectedDriverGUIDs: string[] = [];
        private SelectedChampionshipClassIDs: object = {};

        public constructor($elem: JQuery<HTMLElement>, parentSessionID: string, childSessionID: string) {
            super($elem);

            this.parentSessionID = parentSessionID;
            this.childSessionID = childSessionID;

            this.updateValues();
            this.registerEvents();
        }

        private packageValues(): string {
            return JSON.stringify({
                ResultStart: this.resultStart,
                ResultEnd: this.resultEnd,
                NumEntrantsToReverse: this.reverseGrid,
                EntryListStart: this.gridStart,
                SortType: this.sortType,
                ForceUseTyreFromFastestLap: this.startOnFastestLapTyre,
                AvailableResultsForSorting: this.availableResultsForSorting,
                SplitType: this.splitType,
                SelectedDriverGUIDs: this.selectedDriverGUIDs,
                SelectedChampionshipClassIDs: this.SelectedChampionshipClassIDs,
            })
        }

        protected updateValues(): void {
            this.resultStart = parseInt(this.$elem.find("#ResultsStart").val() as string);
            this.resultEnd = parseInt(this.$elem.find("#ResultsEnd").val() as string);
            this.reverseGrid = parseInt(this.$elem.find("#ReverseGrid").val() as string);
            this.gridStart = parseInt(this.$elem.find("#GridStart").val() as string);
            this.sortType = this.$elem.find("#ResultsSort").val() as string;
            this.availableResultsForSorting = this.$elem.find("#AvailableResults").val() as string[];
            this.startOnFastestLapTyre = this.$elem.find("#ForceUseTyreFromFastestLap").is(":checked");

            if (this.sortType == "fastest_multi_results_lap" || this.sortType == "number_multi_results_lap") {
                this.$elem.find("#AvailableResultsWrapper").show()
            } else {
                this.$elem.find("#AvailableResultsWrapper").hide()
            }

            let $driversMultiSelect = this.$elem.find("#Drivers");
            let $classesMultiSelect = this.$elem.find("#Classes");

            this.splitType = this.$elem.find("#SplitType").val() as SplitType;
            this.selectedDriverGUIDs = $driversMultiSelect.val() as string[];

            this.SelectedChampionshipClassIDs = {};

            for (let classID of $classesMultiSelect.val() as string[]) {
                this.SelectedChampionshipClassIDs[classID] = true;
            }

            switch (this.splitType) {
                case SplitType.Numeric:
                    this.$elem.find("#DriverSelectionForm").hide();
                    this.$elem.find("#ClassSelectionForm").hide();
                    this.$elem.find("#FilterFromTo").show();

                    break;
                case SplitType.ManualDriverSelection:
                    this.$elem.find("#DriverSelectionForm").show();
                    this.$elem.find("#ClassSelectionForm").hide();
                    this.$elem.find("#FilterFromTo").hide();

                    initMultiSelect($driversMultiSelect);
                    break;
                case SplitType.ChampionshipClass:
                    this.$elem.find("#DriverSelectionForm").hide();
                    this.$elem.find("#FilterFromTo").hide();
                    this.$elem.find("#ClassSelectionForm").show();
                    initMultiSelect($classesMultiSelect);
                    break;
            }

            $.ajax(`/race-weekend/${RaceWeekendID}/grid-preview?parentSessionID=${this.parentSessionID}&childSessionID=${this.childSessionID}`, {
                data: this.packageValues(),

                contentType: "application/json",
                type: "POST",
            }).then((response: GridPreview) => {
                let results: SessionPreviewEntrant[] = [];
                let grid: SessionPreviewEntrant[] = [];

                for (const [key, value] of Object.entries(response.Results)) {
                    results.push(value);
                }

                for (const [key, value] of Object.entries(response.Grid)) {
                    grid.push(value);
                }

                let $table = $("table#grid-preview");
                $table.find("tr:not(:first-child)").remove();
                this.buildClassKey(response.Classes);

                for (let i = 0; i < Math.max(grid.length, results.length); i++) {
                    let $row = $("<tr>");

                    if (i < results.length) {
                        $row.append(this.buildTableDataForEntrant(results[i], i));
                    } else {
                        $row.append($("<td>"));
                    }

                    if (i < grid.length) {
                        $row.append(this.buildTableDataForEntrant(grid[i], i));
                    } else {
                        $row.append($("<td>"));
                    }

                    $table.append($row);
                }
            });
        }

        protected saveValues(): void {
            $.ajax(`/race-weekend/${RaceWeekendID}/update-grid?parentSessionID=${this.parentSessionID}&childSessionID=${this.childSessionID}`, {
                data: this.packageValues(),

                contentType: "application/json",
                type: "POST",
            }).then(() => {
                $("#filters-modal").modal("hide");
            });
        }
    }

    interface SessionPreviewEntrant {
        Name: string;
        Session: string;
        Class: string;
        ClassColor: string;
    }

    interface GridPreview {
        Grid: Map<number, SessionPreviewEntrant>;
        Results: Map<number, SessionPreviewEntrant>;
        Classes: Map<string, string>;
    }

    /**
     * EntryListPreview handles the modal which is used to preview and edit the Race Weekend Session final entry list
     */
    class EntryListPreview extends PreviewModal {
        private readonly sessionID: string;
        private sortType: string = "";
        private reverseGrid: number = 0;

        public constructor($elem: JQuery<HTMLElement>, sessionID: string) {
            super($elem);

            this.sessionID = sessionID;

            this.updateValues();
            this.registerEvents();
        }

        protected updateValues(): void {
            this.sortType = this.$elem.find("#SortType").val() as string;
            this.reverseGrid = parseInt(this.$elem.find("#ReverseGrid").val() as string);

            $.ajax(`/race-weekend/${RaceWeekendID}/entrylist-preview?sessionID=${this.sessionID}&sortType=${this.sortType}&reverseGrid=${this.reverseGrid}`, {
                type: "GET",
            }).then((response: GridPreview) => {
                let grid: SessionPreviewEntrant[] = [];

                for (const [key, value] of Object.entries(response.Grid)) {
                    grid.push(value);
                }

                let $table = $("table#entrylist-preview");
                $table.find("tr:not(:first-child)").remove();
                this.buildClassKey(response.Classes);

                for (let i = 0; i < grid.length; i++) {
                    let $row = $("<tr>");
                    let $pos = $("<td>").text(i + 1);

                    if (grid[i].Class) {
                        $pos.css({"background-color": grid[i].ClassColor, "color": "white"});
                    }

                    $row.append($pos);
                    $row.append(this.buildTableDataForEntrant(grid[i]));

                    $table.append($row);
                }
            });
        }

        protected saveValues(): void {
            $.ajax(`/race-weekend/${RaceWeekendID}/update-entrylist?sessionID=${this.sessionID}&sortType=${this.sortType}&reverseGrid=${this.reverseGrid}`, {
                type: "GET",
            }).then(() => {
                $("#filters-modal").modal("hide");
            });
        }
    }
}
