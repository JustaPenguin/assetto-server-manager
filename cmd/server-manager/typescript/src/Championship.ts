import dragula from "dragula";

declare var ChampionshipID: string;
declare var CanMoveChampionshipEvents: boolean;

export namespace Championship {
    export class View {
        public constructor() {
            this.initDraggableCards();
        }

        private initDraggableCards(): void {
            let drake = dragula([document.querySelector(".championship-events")!], {
                moves: (el?: Element, source?: Element, handle?: Element, sibling?: Element): boolean => {
                    if (!CanMoveChampionshipEvents || !handle) {
                        return false;
                    }

                    return $(handle).hasClass("card-header");
                },
            });

            drake.on("drop", () => {
                this.saveChampionshipEventOrder();
            });
        }

        private saveChampionshipEventOrder(): void {
            let championshipEventIDs: string[] = [];

            $(".championship-event").each(function() {
                if (!$(this).hasClass("gu-mirror")) {
                    // dragula duplicates the element being moved as a 'mirror',
                    // ignore it when building championship event id list
                    championshipEventIDs.push($(this).attr("id")!);
                }
            });

            $.ajax({
                type: "POST",
                url: `/championship/${ChampionshipID}/reorder-events`,
                data: JSON.stringify(championshipEventIDs),
                dataType: "json"
            });
        }
    }
}
