// this file was automatically generated, DO NOT EDIT
import {ToObject, FromArray, ParseDate, ParseNumber} from "./helpers";
// classes
// struct2ts:github.com/cj123/assetto-server-manager/pkg/udp.SessionInfo
class SessionInfo {
	Version: number;
	SessionIndex: number;
	CurrentSessionIndex: number;
	SessionCount: number;
	ServerName: string;
	Track: string;
	TrackConfig: string;
	Name: string;
	SessionType: number;
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
		this.SessionType = ('SessionType' in d) ? d.SessionType as number : 0;
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
		cfg.SessionType = 'number';
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

// struct2ts:github.com/cj123/assetto-server-manager/pkg/udp.CarUpdateVec
class CarUpdateVec {
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

// struct2ts:github.com/cj123/assetto-server-manager/pkg/udp.CarUpdate
class CarUpdate {
	CarID: number;
	Pos: CarUpdateVec;
	Velocity: CarUpdateVec;
	Gear: number;
	EngineRPM: number;
	NormalisedSplinePos: number;

	constructor(data?: any) {
		const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
		this.CarID = ('CarID' in d) ? d.CarID as number : 0;
		this.Pos = new CarUpdateVec(d.Pos);
		this.Velocity = new CarUpdateVec(d.Velocity);
		this.Gear = ('Gear' in d) ? d.Gear as number : 0;
		this.EngineRPM = ('EngineRPM' in d) ? d.EngineRPM as number : 0;
		this.NormalisedSplinePos = ('NormalisedSplinePos' in d) ? d.NormalisedSplinePos as number : 0;
	}

	toObject(): any {
		const cfg: any = {};
		cfg.CarID = 'number';
		cfg.Gear = 'number';
		cfg.EngineRPM = 'number';
		cfg.NormalisedSplinePos = 'number';
		return ToObject(this, cfg);
	}
}

// struct2ts:github.com/cj123/assetto-server-manager/pkg/udp.LapCompletedLapCompletedCar
class LapCompletedLapCompletedCar {
	CarID: number;
	LapTime: number;
	Laps: number;
	Completed: number;

	constructor(data?: any) {
		const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
		this.CarID = ('CarID' in d) ? d.CarID as number : 0;
		this.LapTime = ('LapTime' in d) ? d.LapTime as number : 0;
		this.Laps = ('Laps' in d) ? d.Laps as number : 0;
		this.Completed = ('Completed' in d) ? d.Completed as number : 0;
	}

	toObject(): any {
		const cfg: any = {};
		cfg.CarID = 'number';
		cfg.LapTime = 'number';
		cfg.Laps = 'number';
		cfg.Completed = 'number';
		return ToObject(this, cfg);
	}
}

// struct2ts:github.com/cj123/assetto-server-manager/pkg/udp.LapCompleted
class LapCompleted {
	CarID: number;
	LapTime: number;
	Cuts: number;
	CarsCount: number;
	Cars: LapCompletedLapCompletedCar[];

	constructor(data?: any) {
		const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
		this.CarID = ('CarID' in d) ? d.CarID as number : 0;
		this.LapTime = ('LapTime' in d) ? d.LapTime as number : 0;
		this.Cuts = ('Cuts' in d) ? d.Cuts as number : 0;
		this.CarsCount = ('CarsCount' in d) ? d.CarsCount as number : 0;
		this.Cars = Array.isArray(d.Cars) ? d.Cars.map((v: any) => new LapCompletedLapCompletedCar(v)) : [];
	}

	toObject(): any {
		const cfg: any = {};
		cfg.CarID = 'number';
		cfg.LapTime = 'number';
		cfg.Cuts = 'number';
		cfg.CarsCount = 'number';
		return ToObject(this, cfg);
	}
}

// struct2ts:github.com/cj123/assetto-server-manager/pkg/udp.CollisionWithEnvironment
class CollisionWithEnvironment {
	CarID: number;
	ImpactSpeed: number;
	WorldPos: CarUpdateVec;
	RelPos: CarUpdateVec;

	constructor(data?: any) {
		const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
		this.CarID = ('CarID' in d) ? d.CarID as number : 0;
		this.ImpactSpeed = ('ImpactSpeed' in d) ? d.ImpactSpeed as number : 0;
		this.WorldPos = new CarUpdateVec(d.WorldPos);
		this.RelPos = new CarUpdateVec(d.RelPos);
	}

	toObject(): any {
		const cfg: any = {};
		cfg.CarID = 'number';
		cfg.ImpactSpeed = 'number';
		return ToObject(this, cfg);
	}
}

// struct2ts:github.com/cj123/assetto-server-manager/pkg/udp.CollisionWithCar
class CollisionWithCar {
	CarID: number;
	OtherCarID: number;
	ImpactSpeed: number;
	WorldPos: CarUpdateVec;
	RelPos: CarUpdateVec;

	constructor(data?: any) {
		const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
		this.CarID = ('CarID' in d) ? d.CarID as number : 0;
		this.OtherCarID = ('OtherCarID' in d) ? d.OtherCarID as number : 0;
		this.ImpactSpeed = ('ImpactSpeed' in d) ? d.ImpactSpeed as number : 0;
		this.WorldPos = new CarUpdateVec(d.WorldPos);
		this.RelPos = new CarUpdateVec(d.RelPos);
	}

	toObject(): any {
		const cfg: any = {};
		cfg.CarID = 'number';
		cfg.OtherCarID = 'number';
		cfg.ImpactSpeed = 'number';
		return ToObject(this, cfg);
	}
}

// struct2ts:github.com/cj123/assetto-server-manager/pkg/udp.Chat
class Chat {
	CarID: number;
	Message: string;

	constructor(data?: any) {
		const d: any = (data && typeof data === 'object') ? ToObject(data) : {};
		this.CarID = ('CarID' in d) ? d.CarID as number : 0;
		this.Message = ('Message' in d) ? d.Message as string : '';
	}

	toObject(): any {
		const cfg: any = {};
		cfg.CarID = 'number';
		return ToObject(this, cfg);
	}
}

// exports
export {
	SessionInfo,
	CarUpdateVec,
	CarUpdate,
	LapCompletedLapCompletedCar,
	LapCompleted,
	CollisionWithEnvironment,
	CollisionWithCar,
	Chat,
};
