import { MapContainer } from 'react-leaflet/MapContainer'
import { TileLayer } from 'react-leaflet/TileLayer'
import './App.css'
import {useEffect, useState} from "react";
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

function App() {
    const [datapoints, setDatapoints] = useState<Datapoint[]>([]);

    const [groupedDatapoints, setGroupedDatapoints] = useState<Record<string, Datapoint[]>>({});

    const [groupByTeam, setGroupByTeam] = useState<boolean>(true);

    const [selectedEntities, setSelectedEntities] = useState<Record<string, boolean>>({});

    const [teamMapping, setTeamMapping] = useState<Record<string, { name: string }>>({});

    const [enableLabels, setEnableLabels] = useState<boolean>(true);

    function fetchTracks() {
        fetch("/api/tracks")
            .then(res => res.json())
            .then(data => {
                setDatapoints(data.map((d: any) => {
                    return {
                        team: d.team,
                        user: d.user,
                        teamName: d.user,
                        latitude: d.latitude,
                        longitude: d.longitude,
                        battery: d.battery,
                        time: new Date(d.timestamp*1000),
                        distanceLeft: distance([d.longitude, d.latitude] , [5.4848275, 51.4468853], /*[9.9099321, 53.5165228] */ )
                    }
                }))
            })
    }

    function fetchTeamMapping() {
        fetch("/teams.json")
            .then(res => res.json())
            .then(data => {
                setTeamMapping(data);
            })
    }

    function getTeamName(id: string) {
        return teamMapping[id]?.name || id;
    }

    useEffect(() => {
        fetchTracks();

        fetchTeamMapping();

        const interval = setInterval(fetchTracks, 10*1000);

        return () => clearInterval(interval);
    }, [])

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
            { /* Overlay */ }
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
            <MapContainer className={"map-container z-10"} center={[52.278,7.973]} zoom={8}>
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

export default App
