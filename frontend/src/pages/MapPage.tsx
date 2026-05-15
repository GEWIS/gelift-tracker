import { MapContainer } from 'react-leaflet/MapContainer'
import { TileLayer } from 'react-leaflet/TileLayer'
import '../App.css'
import {useEffect, useMemo, useState} from "react";
import {CircleMarker, Polyline, Popup, Tooltip} from "react-leaflet";
import uniqolor from "uniqolor";
import { Panel } from 'primereact/panel';
import {InputSwitch} from "primereact/inputswitch";
import distance from "@turf/distance"
import {Checkbox} from "primereact/checkbox";
import {Divider} from "primereact/divider";

interface Datapoint {
    team: string;
    user: string;
    teamName: string;
    latitude: number;
    longitude: number;
    battery: number;
    time: Date;
    distanceLeft: number;
}

/** Fallback when `finish_latitude` / `finish_longitude` settings are not set (lon, lat for @turf/distance). */
const DEFAULT_FINISH: { lng: number; lat: number } = { lng: 6.703752, lat: 52.2992009 }

interface RawTrackRow {
    team: string
    user: string
    latitude: number
    longitude: number
    battery: number
    timestamp: number
}

function getColor(name: string) {
    return uniqolor(name, { lightness: [35, 50] }).color
}

function getDistanceOrTime(datapoints: Datapoint[]): string {
    const datapointAtDest = datapoints.filter(d => d.distanceLeft < 1);

    if (datapointAtDest.length === 0) {
        return `${Math.round(datapoints[datapoints.length-1].distanceLeft)} km left`;
    } else {
        return `Arrived at ${datapointAtDest[0].time.toLocaleTimeString("nl-NL")}`;
    }
}

function formatDurationRemaining(remainingMs: number): string {
    if (remainingMs <= 0) return '0:00:00'
    const totalSec = Math.ceil(remainingMs / 1000)
    const days = Math.floor(totalSec / 86400)
    const h = Math.floor((totalSec % 86400) / 3600)
    const m = Math.floor((totalSec % 3600) / 60)
    const s = totalSec % 60
    const hm = `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
    return days > 0 ? `${days}d ${hm}` : hm
}

export function MapPage() {
    const [rawTracks, setRawTracks] = useState<RawTrackRow[]>([])

    const [finish, setFinish] = useState<{ lng: number; lat: number }>(DEFAULT_FINISH)

    const [groupedDatapoints, setGroupedDatapoints] = useState<Record<string, Datapoint[]>>({});

    const [groupByTeam, setGroupByTeam] = useState<boolean>(true);

    const [selectedEntities, setSelectedEntities] = useState<Record<string, boolean>>({});

    const [teamMapping, setTeamMapping] = useState<Record<string, { name: string }>>({});

    const [enableLabels, setEnableLabels] = useState<boolean>(true);

    const [settings, setSettings] = useState<Record<string, string>>({});

    const [startAtMs, setStartAtMs] = useState<number | null>(null)
    const [countdownTick, setCountdownTick] = useState(0)

    const datapoints = useMemo((): Datapoint[] => {
        return rawTracks.map((d) => ({
            team: d.team,
            user: d.user,
            teamName: d.user,
            latitude: d.latitude,
            longitude: d.longitude,
            battery: d.battery,
            time: new Date(d.timestamp * 1000),
            distanceLeft: distance([d.longitude, d.latitude], [finish.lng, finish.lat], { units: 'kilometers' }),
        }))
    }, [rawTracks, finish])

    function fetchTracks() {
        fetch("/api/tracks")
            .then(res => res.json())
            .then((data: unknown) => {
                if (!Array.isArray(data)) {
                    setRawTracks([])
                    return
                }
                setRawTracks(
                    data.map((d: any) => ({
                        team: d.team,
                        user: d.user,
                        latitude: d.latitude,
                        longitude: d.longitude,
                        battery: d.battery,
                        timestamp: d.timestamp,
                    })),
                )
            })
    }

    function fetchTeamMapping() {
        fetch("/teams.json")
            .then(res => res.json())
            .then(data => {
                setTeamMapping(data);
            })
    }

    function fetchSettings() {
        fetch("/api/settings")
            .then(res => res.json())
            .then(data => {
                let _settings: Record<string, string> = {}
                for(let setting of data) {
                    _settings[setting.key as string] = setting.value
                }
                setSettings(_settings);
            })
    }

    function getTeamName(id: string) {
        return teamMapping[id]?.name || id;
    }

    useEffect(() => {
        fetchSettings();

        fetchTracks();

        fetchTeamMapping();

        const interval = setInterval(fetchTracks, 10*1000);

        return () => clearInterval(interval);
    }, [])

    useEffect(() => {
        if (settings['start_time']) {
            setStartAtMs(Date.parse(settings['start_time']))
        }

        if (settings['finish_latitude'] && settings['finish_longitude']) {
            setFinish({
                lng: Number(settings['finish_longitude']),
                lat: Number(settings['finish_latitude'])
            });
        }
    }, [settings])

    useEffect(() => {
        if (startAtMs === null) return undefined
        const id = setInterval(() => setCountdownTick((n) => n + 1), 1000)
        return () => clearInterval(id)
    }, [startAtMs])

    const beforeStart = useMemo(() => {
        return startAtMs !== null && Date.now() < startAtMs
    }, [startAtMs, countdownTick])

    const remainingMs = useMemo(() => {
        if (startAtMs === null) return 0
        return Math.max(0, startAtMs - Date.now())
    }, [startAtMs, countdownTick])

    useEffect(() => {
        const uniqueEntities = new Set<string>();

        for (const point of datapoints) {
            if (groupByTeam) {
                uniqueEntities.add(point.team);
            } else {
                uniqueEntities.add(point.user);
            }
        }

        const _groupedDatapoints: Record<string, Datapoint[]> = {};
        const _selectedEntities: Record<string, boolean> = {};

        for (const uniqueEntity of uniqueEntities) {
            _selectedEntities[uniqueEntity] = selectedEntities[uniqueEntity] === undefined
                ? true
                : selectedEntities[uniqueEntity]
            _groupedDatapoints[uniqueEntity] = datapoints.filter(d => {
                return groupByTeam && d.team === uniqueEntity
                    || d.user === uniqueEntity
            });
        }

        setSelectedEntities(_selectedEntities);
        setGroupedDatapoints(_groupedDatapoints);
    }, [datapoints, groupByTeam])

    return (
        <>
            {beforeStart && startAtMs !== null && (
                <div
                    className="fixed inset-0 z-[2000] flex flex-col items-center justify-center gap-4 bg-slate-900 text-white px-6 text-center"
                    role="status"
                    aria-live="polite"
                >
                    <p className="m-0 text-lg font-medium text-slate-200">GELIFT 2026 starts in</p>
                    <p className="m-0 text-5xl font-semibold tabular-nums tracking-tight sm:text-6xl">
                        {formatDurationRemaining(remainingMs)}
                    </p>
                </div>
            )}
            <div className={"absolute flex flex-row justify-end w-full"}>
                <div className={"flex flex-col w-2xs"}>
                    <Panel header={"Controls"} className={"bg-white flex flex-col m-1 z-500 bg rounded"} toggleable collapsed={true}>
                        <div className={"flex flex-row mb-2"}>
                            <InputSwitch
                                checked={!groupByTeam}
                                onChange={(e) => setGroupByTeam(!e.value)}
                            />
                            <div className={"ml-2"}>Individual view</div>
                        </div>
                        <div className={"flex flex-row"}>
                            <InputSwitch
                                checked={!enableLabels}
                                onChange={(e) => setEnableLabels(!e.value)}
                            />
                            <div className={"ml-2"}>Disable team labels</div>
                        </div>
                    </Panel>
                    <Panel header={"Teams"} className={"bg-white flex flex-col mx-1 z-500 bg rounded"} toggleable collapsed={window.innerWidth < 500}>
                        <div className={"grid grid-cols-5 justify-items-start"}>
                            { Object.values(groupedDatapoints)
                                .sort((a, b) => {
                                    a.sort((c, d) => c.time.getTime()-d.time.getTime())
                                    b.sort((e, f) => e.time.getTime()-f.time.getTime())

                                    const datapointAtDestA = a.filter(d => d.distanceLeft < 1);
                                    const datapointAtDestB = b.filter(d => d.distanceLeft < 1);

                                    if (datapointAtDestB.length === 0 && datapointAtDestA.length === 0) {
                                        return a[a.length-1].distanceLeft - b[b.length-1].distanceLeft
                                    } else if (datapointAtDestA.length === 0 && datapointAtDestB.length > 0) {
                                        return 1
                                    } else if (datapointAtDestB.length === 0 && datapointAtDestA.length > 0) {
                                        return -1
                                    } else {
                                        return datapointAtDestA[0].time.getTime() - datapointAtDestB[0].time.getTime();
                                    }

                                })
                                .map((p, i) => {
                                    p.sort((a, b) => a.time.getTime()-b.time.getTime())
                                    const lastPos = p[p.length-1]
                                    const color = getColor(groupByTeam ? lastPos.team : lastPos.user)
                                    return <>
                                        <Checkbox
                                            checked={selectedEntities[groupByTeam ? lastPos.team : lastPos.user]}
                                            onChange={e => {
                                                setSelectedEntities({
                                                    ...selectedEntities,
                                                    [groupByTeam ? lastPos.team : lastPos.user]: e.checked!,
                                                })
                                            }}
                                        />
                                        <div className={"col-span-2 text-start"} style={{ color }}>
                                            { groupByTeam ? getTeamName(lastPos.team) : lastPos.user }
                                        </div>
                                        <div className={"col-span-2"}>
                                            { getDistanceOrTime(p) }
                                        </div>
                                        { i !== Object.values(groupedDatapoints).length-1
                                            && <Divider className={"col-span-5 m-0"}/> }
                                    </>
                                })
                            }
                        </div>
                    </Panel>
                </div>
            </div>
            <MapContainer className={"map-container z-10"} center={[50.624,5.516]} zoom={7}>
                <TileLayer
                    attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                    url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                />
                {
                    Object.values(groupedDatapoints).map((p) => {
                        p.sort((a, b) => a.time.getTime()-b.time.getTime())
                        const lastPos = p[p.length-1]
                        if (!selectedEntities[groupByTeam ? lastPos.team : lastPos.user]) {
                            return <></>
                        }
                        const color = getColor(groupByTeam ? lastPos.team : lastPos.user)
                        return <>
                            <CircleMarker center={[lastPos.latitude, lastPos.longitude]} pathOptions={{ color }} radius={10}>
                                <Popup>
                                    <b>Last position details:</b> <br/>
                                    Time: { lastPos.time.toLocaleString() } <br/>
                                    Reported by: { lastPos.user } <br/>
                                    Battery: { lastPos.battery } <br/>
                                </Popup>
                                {
                                    enableLabels &&
                                    <Tooltip direction="bottom" offset={[0, 10]} opacity={1} permanent>
                                        { groupByTeam ? getTeamName(lastPos.team) : lastPos.user }
                                    </Tooltip>
                                }
                            </CircleMarker>
                            <Polyline
                                pathOptions={{ color }}
                                positions={p.map(d => [d.latitude, d.longitude])} />
                        </>
                    })
                }
            </MapContainer>
        </>
    )
}
