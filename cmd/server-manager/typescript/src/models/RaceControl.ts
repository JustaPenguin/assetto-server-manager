// this file was automatically generated, DO NOT EDIT

// helpers
const maxUnixTSInSeconds = 9999999999;

function ParseDate(d: Date | number | string): Date {
	if (d instanceof Date) return d;
	if (typeof d === 'number') {
		if (d > maxUnixTSInSeconds) return new Date(d);
		return new Date(d * 1000); // go ts
	}
	return new Date(d);
}

function ParseNumber(v: number | string, isInt = false): number {
	if (!v) return 0;
	if (typeof v === 'number') return v;
	return (isInt ? parseInt(v) : parseFloat(v)) || 0;
}

function FromArray<T>(Ctor: { new(v: any): T }, data?: any[] | any, def = null): T[] | null {
	if (!data || !Object.keys(data).length) return def;
	const d = Array.isArray(data) ? data : [data];
	return d.map((v: any) => new Ctor(v));
}

function ToObject(o: any, typeOrCfg: any = {}, child = false): any {
	if (!o) return null;
	if (typeof o.toObject === 'function' && child) return o.toObject();

	switch (typeof o) {
		case 'string':
			return typeOrCfg === 'number' ? ParseNumber(o) : o;
		case 'boolean':
		case 'number':
			return o;
	}

	if (o instanceof Date) {
		return typeOrCfg === 'string' ? o.toISOString() : Math.floor(o.getTime() / 1000);
	}

	if (Array.isArray(o)) return o.map((v: any) => ToObject(v, typeOrCfg, true));

	const d: any = {};

	for (const k of Object.keys(o)) {
		const v: any = o[k];
		if (!v) continue;
		d[k] = ToObject(v, typeOrCfg[k] || {}, true);
	}

	return d;
}

// classes
// struct2ts:github.com/JustaPenguin/assetto-server-manager/pkg/udp.RaceControlSessionInfo
class RaceControlSessionInfo {
    Version: number;
    SessionIndex: number;
    CurrentSessionIndex: number;
    SessionCount: number;
    ServerName: string;
    Track: string;
    TrackConfig: string;
    Name: string;
    Type: number;
    Time: number;
    Laps: number;
    WaitTime: number;
    AmbientTemp: number;
    RoadTemp: number;
    WeatherGraphics: string;
    ElapsedMilliseconds: number;
    EventType: number;

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.Version = ('Version' in d) ? d.Version as number : 0;
        this.SessionIndex = ('SessionIndex' in d) ? d.SessionIndex as number : 0;
        this.CurrentSessionIndex = ('CurrentSessionIndex' in d) ? d.CurrentSessionIndex as number : 0;
        this.SessionCount = ('SessionCount' in d) ? d.SessionCount as number : 0;
        this.ServerName = ('ServerName' in d) ? d.ServerName as string : '';
        this.Track = ('Track' in d) ? d.Track as string : '';
        this.TrackConfig = ('TrackConfig' in d) ? d.TrackConfig as string : '';
        this.Name = ('Name' in d) ? d.Name as string : '';
        this.Type = ('Type' in d) ? d.Type as number : 0;
        this.Time = ('Time' in d) ? d.Time as number : 0;
        this.Laps = ('Laps' in d) ? d.Laps as number : 0;
        this.WaitTime = ('WaitTime' in d) ? d.WaitTime as number : 0;
        this.AmbientTemp = ('AmbientTemp' in d) ? d.AmbientTemp as number : 0;
        this.RoadTemp = ('RoadTemp' in d) ? d.RoadTemp as number : 0;
        this.WeatherGraphics = ('WeatherGraphics' in d) ? d.WeatherGraphics as string : '';
        this.ElapsedMilliseconds = ('ElapsedMilliseconds' in d) ? d.ElapsedMilliseconds as number : 0;
        this.EventType = ('EventType' in d) ? d.EventType as number : 0;
    }

    toObject(): any {
        const cfg: any = {};
        cfg.Version = 'number';
        cfg.SessionIndex = 'number';
        cfg.CurrentSessionIndex = 'number';
        cfg.SessionCount = 'number';
        cfg.Type = 'number';
        cfg.Time = 'number';
        cfg.Laps = 'number';
        cfg.WaitTime = 'number';
        cfg.AmbientTemp = 'number';
        cfg.RoadTemp = 'number';
        cfg.ElapsedMilliseconds = 'number';
        cfg.EventType = 'number';
        return ToObject(this, cfg);
    }
}

// struct2ts:github.com/JustaPenguin/assetto-server-manager.RaceControlTrackMapData
class RaceControlTrackMapData {
    width: number;
    height: number;
    margin: number;
    scale_factor: number;
    offset_x: number;
    offset_y: number;
    drawing_size: number;

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.width = ('width' in d) ? d.width as number : 0;
        this.height = ('height' in d) ? d.height as number : 0;
        this.margin = ('margin' in d) ? d.margin as number : 0;
        this.scale_factor = ('scale_factor' in d) ? d.scale_factor as number : 0;
        this.offset_x = ('offset_x' in d) ? d.offset_x as number : 0;
        this.offset_y = ('offset_y' in d) ? d.offset_y as number : 0;
        this.drawing_size = ('drawing_size' in d) ? d.drawing_size as number : 0;
    }

    toObject(): any {
        const cfg: any = {};
        cfg.width = 'number';
        cfg.height = 'number';
        cfg.margin = 'number';
        cfg.scale_factor = 'number';
        cfg.offset_x = 'number';
        cfg.offset_y = 'number';
        cfg.drawing_size = 'number';
        return ToObject(this, cfg);
    }
}

// struct2ts:github.com/JustaPenguin/assetto-server-manager.RaceControlTrackInfo
class RaceControlTrackInfo {
    name: string;
    city: string;
    country: string;
    description: string;
    geotags: string[];
    length: string;
    pitboxes: string;
    run: string;
    tags: string[];
    width: string;

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.name = ('name' in d) ? d.name as string : '';
        this.city = ('city' in d) ? d.city as string : '';
        this.country = ('country' in d) ? d.country as string : '';
        this.description = ('description' in d) ? d.description as string : '';
        this.geotags = ('geotags' in d) ? d.geotags as string[] : [];
        this.length = ('length' in d) ? d.length as string : '';
        this.pitboxes = ('pitboxes' in d) ? d.pitboxes as string : '';
        this.run = ('run' in d) ? d.run as string : '';
        this.tags = ('tags' in d) ? d.tags as string[] : [];
        this.width = ('width' in d) ? d.width as string : '';
    }

    toObject(): any {
        const cfg: any = {};
        return ToObject(this, cfg);
    }
}

// struct2ts:github.com/JustaPenguin/assetto-server-manager/pkg/udp.RaceControlDriverMapRaceControlDriverSessionCarInfo
class RaceControlDriverMapRaceControlDriverSessionCarInfo {
    CarID: number;
    DriverName: string;
    DriverGUID: string;
    CarModel: string;
    CarSkin: string;
    DriverInitials: string;
    CarName: string;
    EventType: number;

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.CarID = ('CarID' in d) ? d.CarID as number : 0;
        this.DriverName = ('DriverName' in d) ? d.DriverName as string : '';
        this.DriverGUID = ('DriverGUID' in d) ? d.DriverGUID as string : '';
        this.CarModel = ('CarModel' in d) ? d.CarModel as string : '';
        this.CarSkin = ('CarSkin' in d) ? d.CarSkin as string : '';
        this.DriverInitials = ('DriverInitials' in d) ? d.DriverInitials as string : '';
        this.CarName = ('CarName' in d) ? d.CarName as string : '';
        this.EventType = ('EventType' in d) ? d.EventType as number : 0;
    }

    toObject(): any {
        const cfg: any = {};
        cfg.CarID = 'number';
        cfg.EventType = 'number';
        return ToObject(this, cfg);
    }
}

// struct2ts:github.com/JustaPenguin/assetto-server-manager/pkg/udp.RaceControlDriverMapRaceControlDriverVec
class RaceControlDriverMapRaceControlDriverVec {
    X: number;
    Y: number;
    Z: number;

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.X = ('X' in d) ? d.X as number : 0;
        this.Y = ('Y' in d) ? d.Y as number : 0;
        this.Z = ('Z' in d) ? d.Z as number : 0;
    }

    toObject(): any {
        const cfg: any = {};
        cfg.X = 'number';
        cfg.Y = 'number';
        cfg.Z = 'number';
        return ToObject(this, cfg);
    }
}

// struct2ts:github.com/JustaPenguin/assetto-server-manager.RaceControlDriverMapRaceControlDriverCollision
class RaceControlDriverMapRaceControlDriverCollision {
    ID: string;
    Type: string;
    Time: Date;
    OtherDriverGUID: string;
    OtherDriverName: string;
    Speed: number;

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.ID = ('ID' in d) ? d.ID as string : '';
        this.Type = ('Type' in d) ? d.Type as string : '';
        this.Time = ('Time' in d) ? ParseDate(d.Time) : new Date();
        this.OtherDriverGUID = ('OtherDriverGUID' in d) ? d.OtherDriverGUID as string : '';
        this.OtherDriverName = ('OtherDriverName' in d) ? d.OtherDriverName as string : '';
        this.Speed = ('Speed' in d) ? d.Speed as number : 0;
    }

    toObject(): any {
        const cfg: any = {};
        cfg.Time = 'string';
        cfg.Speed = 'number';
        return ToObject(this, cfg);
    }
}

// struct2ts:github.com/JustaPenguin/assetto-server-manager.RaceControlDriverMapRaceControlDriverRaceControlCarLapInfo
class RaceControlDriverMapRaceControlDriverRaceControlCarLapInfo {
    TopSpeedThisLap: number;
    TopSpeedBestLap: number;
    BestLap: number;
    NumLaps: number;
    LastLap: number;
    LastLapCompletedTime: Date;
    TotalLapTime: number;
    CarName: string;

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.TopSpeedThisLap = ('TopSpeedThisLap' in d) ? d.TopSpeedThisLap as number : 0;
        this.TopSpeedBestLap = ('TopSpeedBestLap' in d) ? d.TopSpeedBestLap as number : 0;
        this.BestLap = ('BestLap' in d) ? d.BestLap as number : 0;
        this.NumLaps = ('NumLaps' in d) ? d.NumLaps as number : 0;
        this.LastLap = ('LastLap' in d) ? d.LastLap as number : 0;
        this.LastLapCompletedTime = ('LastLapCompletedTime' in d) ? ParseDate(d.LastLapCompletedTime) : new Date();
        this.TotalLapTime = ('TotalLapTime' in d) ? d.TotalLapTime as number : 0;
        this.CarName = ('CarName' in d) ? d.CarName as string : '';
    }

    toObject(): any {
        const cfg: any = {};
        cfg.TopSpeedThisLap = 'number';
        cfg.TopSpeedBestLap = 'number';
        cfg.BestLap = 'number';
        cfg.NumLaps = 'number';
        cfg.LastLap = 'number';
        cfg.LastLapCompletedTime = 'string';
        cfg.TotalLapTime = 'number';
        return ToObject(this, cfg);
    }
}

// struct2ts:github.com/JustaPenguin/assetto-server-manager.RaceControlDriverMapRaceControlDriver
class RaceControlDriverMapRaceControlDriver {
    CarInfo: RaceControlDriverMapRaceControlDriverSessionCarInfo;
    TotalNumLaps: number;
    ConnectedTime: Date;
    LoadedTime: Date;
    Position: number;
    Split: string;
    LastSeen: Date;
    LastPos: RaceControlDriverMapRaceControlDriverVec;
    Collisions: RaceControlDriverMapRaceControlDriverCollision[];
    Cars: { [key: string]: RaceControlDriverMapRaceControlDriverRaceControlCarLapInfo };

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.CarInfo = new RaceControlDriverMapRaceControlDriverSessionCarInfo(d.CarInfo);
        this.TotalNumLaps = ('TotalNumLaps' in d) ? d.TotalNumLaps as number : 0;
        this.ConnectedTime = ('ConnectedTime' in d) ? ParseDate(d.ConnectedTime) : new Date();
        this.LoadedTime = ('LoadedTime' in d) ? ParseDate(d.LoadedTime) : new Date();
        this.Position = ('Position' in d) ? d.Position as number : 0;
        this.Split = ('Split' in d) ? d.Split as string : '';
        this.LastSeen = ('LastSeen' in d) ? ParseDate(d.LastSeen) : new Date();
        this.LastPos = new RaceControlDriverMapRaceControlDriverVec(d.LastPos);
        this.Collisions = Array.isArray(d.Collisions) ? d.Collisions.map((v: any) => new RaceControlDriverMapRaceControlDriverCollision(v)) : [];
        this.Cars = ('Cars' in d) ? d.Cars as { [key: string]: RaceControlDriverMapRaceControlDriverRaceControlCarLapInfo } : {};
    }

    toObject(): any {
        const cfg: any = {};
        cfg.TotalNumLaps = 'number';
        cfg.ConnectedTime = 'string';
        cfg.LoadedTime = 'string';
        cfg.Position = 'number';
        cfg.LastSeen = 'string';
        return ToObject(this, cfg);
    }
}

// struct2ts:github.com/JustaPenguin/assetto-server-manager.RaceControlDriverMap
class RaceControlDriverMap {
    Drivers: { [key: string]: RaceControlDriverMapRaceControlDriver };
    GUIDsInPositionalOrder: string[];

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.Drivers = ('Drivers' in d) ? d.Drivers as { [key: string]: RaceControlDriverMapRaceControlDriver } : {};
        this.GUIDsInPositionalOrder = ('GUIDsInPositionalOrder' in d) ? d.GUIDsInPositionalOrder as string[] : [];
    }

    toObject(): any {
        const cfg: any = {};
        return ToObject(this, cfg);
    }
}

// struct2ts:github.com/JustaPenguin/assetto-server-manager.RaceControl
class RaceControl {
    SessionInfo: RaceControlSessionInfo;
    TrackMapData: RaceControlTrackMapData;
    TrackInfo: RaceControlTrackInfo;
    SessionStartTime: Date;
    CurrentRealtimePosInterval: number;
    ConnectedDrivers: RaceControlDriverMap | null;
    DisconnectedDrivers: RaceControlDriverMap | null;
    CarIDToGUID: { [key: number]: string };

    constructor(data?: any) {
        const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
        this.SessionInfo = new RaceControlSessionInfo(d.SessionInfo);
        this.TrackMapData = new RaceControlTrackMapData(d.TrackMapData);
        this.TrackInfo = new RaceControlTrackInfo(d.TrackInfo);
        this.SessionStartTime = ('SessionStartTime' in d) ? ParseDate(d.SessionStartTime) : new Date();
        this.CurrentRealtimePosInterval = ('CurrentRealtimePosInterval' in d) ? d.CurrentRealtimePosInterval as number : 0;
        this.ConnectedDrivers = ('ConnectedDrivers' in d) ? new RaceControlDriverMap(d.ConnectedDrivers) : null;
        this.DisconnectedDrivers = ('DisconnectedDrivers' in d) ? new RaceControlDriverMap(d.DisconnectedDrivers) : null;
        this.CarIDToGUID = ('CarIDToGUID' in d) ? d.CarIDToGUID as { [key: number]: string } : {};
    }

    toObject(): any {
        const cfg: any = {};
        cfg.SessionStartTime = 'string';
        cfg.CurrentRealtimePosInterval = 'number';
        return ToObject(this, cfg);
    }
}

// exports
export {
    RaceControlSessionInfo,
    RaceControlTrackMapData,
    RaceControlTrackInfo,
    RaceControlDriverMapRaceControlDriverSessionCarInfo,
    RaceControlDriverMapRaceControlDriverVec,
    RaceControlDriverMapRaceControlDriverCollision,
    RaceControlDriverMapRaceControlDriverRaceControlCarLapInfo,
    RaceControlDriverMapRaceControlDriver,
    RaceControlDriverMap,
    RaceControl,
    ParseDate,
    ParseNumber,
    FromArray,
    ToObject,
};
