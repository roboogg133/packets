package repo

import (
	"database/sql"

	"github.com/roboogg133/packets/cmd/packets/database"
	"github.com/roboogg133/packets/pkg/packet.lua.d"
)

type DependencyStatus struct {
	Id       packet.PackageID
	Serial   int
	Location string
}

const (
	RuntimeDependenciesQuery = `WITH dependency_info AS (
   SELECT
       d.dependency_name,
       d.version_constraint,
       d.location,
       CASE
           WHEN d.version_constraint = char(0) THEN 'HIGHEST'
           WHEN d.version_constraint LIKE '>=%' THEN 'GREATER_EQUAL'
           WHEN d.version_constraint LIKE '<=%' THEN 'LESS_EQUAL'
           WHEN d.version_constraint LIKE '%@%' THEN 'EXACT'
           ELSE 'EXACT'
       END as constraint_type,
       CASE
           WHEN d.version_constraint = char(0) THEN NULL
           WHEN d.version_constraint LIKE '>=%' THEN SUBSTR(d.version_constraint, 3)
           WHEN d.version_constraint LIKE '<=%' THEN SUBSTR(d.version_constraint, 3)
           WHEN d.version_constraint LIKE '%@%' THEN SUBSTR(d.version_constraint, INSTR(d.version_constraint, '@') + 1)
           ELSE d.version_constraint
       END as constraint_version
   FROM dependencies d
   WHERE d.package_id = ?
),
version_parts AS (
   SELECT
       p.name,
       p.id,
       p.version,
       p.serial,
       p.location,
       CAST(SUBSTR(p.version, 1, INSTR(p.version || '.', '.') - 1) AS INTEGER) as major,
       CASE
           WHEN INSTR(p.version, '.') > 0 THEN
               CAST(SUBSTR(
                   SUBSTR(p.version, INSTR(p.version, '.') + 1),
                   1,
                   INSTR(SUBSTR(p.version, INSTR(p.version, '.') + 1) || '.', '.') - 1
               ) AS INTEGER)
           ELSE 0
       END as minor,
       CASE
           WHEN (LENGTH(p.version) - LENGTH(REPLACE(p.version, '.', ''))) >= 2 THEN
               CAST(SUBSTR(
                   p.version,
                   INSTR(p.version, '.') + INSTR(SUBSTR(p.version, INSTR(p.version, '.') + 1), '.') + 1
               ) AS INTEGER)
           ELSE 0
       END as patch
   FROM packages p
),
constraint_version_parts AS (
   SELECT
       di.dependency_name,
       di.version_constraint,
       di.location,
       di.constraint_type,
       di.constraint_version,
       CAST(SUBSTR(di.constraint_version, 1, INSTR(di.constraint_version || '.', '.') - 1) AS INTEGER) as const_major,
       CASE
           WHEN INSTR(di.constraint_version, '.') > 0 THEN
               CAST(SUBSTR(
                   SUBSTR(di.constraint_version, INSTR(di.constraint_version, '.') + 1),
                   1,
                   INSTR(SUBSTR(di.constraint_version, INSTR(di.constraint_version, '.') + 1) || '.', '.') - 1
               ) AS INTEGER)
           ELSE 0
       END as const_minor,
       CASE
           WHEN (LENGTH(di.constraint_version) - LENGTH(REPLACE(di.constraint_version, '.', ''))) >= 2 THEN
               CAST(SUBSTR(
                   di.constraint_version,
                   INSTR(di.constraint_version, '.') + INSTR(SUBSTR(di.constraint_version, INSTR(di.constraint_version, '.') + 1), '.') + 1
               ) AS INTEGER)
           ELSE 0
       END as const_patch
   FROM dependency_info di
   WHERE di.constraint_version IS NOT NULL
),
matched_packages AS (
   SELECT
       di.dependency_name,
       di.version_constraint,
       di.location,
       di.constraint_type,
       vp.id as package_id,
       vp.version as package_version,
       vp.serial,
       vp.major,
       vp.minor,
       vp.patch,
       RANK() OVER (
           PARTITION BY di.dependency_name, di.location
           ORDER BY vp.major DESC, vp.minor DESC, vp.patch DESC
       ) as version_rank
   FROM dependency_info di
   JOIN version_parts vp ON vp.name = di.dependency_name AND vp.location = di.location
   LEFT JOIN constraint_version_parts cvp ON di.dependency_name = cvp.dependency_name
       AND di.location = cvp.location
       AND di.constraint_type = cvp.constraint_type
   WHERE
       (di.constraint_type = 'HIGHEST')
       OR (di.constraint_type = 'EXACT' AND vp.version = di.constraint_version)
       OR (di.constraint_type = 'GREATER_EQUAL' AND (
           vp.major > cvp.const_major OR
           (vp.major = cvp.const_major AND vp.minor > cvp.const_minor) OR
           (vp.major = cvp.const_major AND vp.minor = cvp.const_minor AND vp.patch >= cvp.const_patch)
       ))
       OR (di.constraint_type = 'LESS_EQUAL' AND (
           vp.major < cvp.const_major OR
           (vp.major = cvp.const_major AND vp.minor < cvp.const_minor) OR
           (vp.major = cvp.const_major AND vp.minor = cvp.const_minor AND vp.patch <= cvp.const_patch)
       ))
)
SELECT
   package_id,
   location,
   package_version,
   serial
FROM matched_packages
WHERE version_rank = 1;`
	BuildDependenciesQuery = `WITH dependency_info AS (
   SELECT
       d.dependency_name,
       d.version_constraint,
       d.location,
       CASE
           WHEN d.version_constraint = char(0) THEN 'HIGHEST'
           WHEN d.version_constraint LIKE '>=%' THEN 'GREATER_EQUAL'
           WHEN d.version_constraint LIKE '<=%' THEN 'LESS_EQUAL'
           WHEN d.version_constraint LIKE '%@%' THEN 'EXACT'
           ELSE 'EXACT'
       END as constraint_type,
       CASE
           WHEN d.version_constraint = char(0) THEN NULL
           WHEN d.version_constraint LIKE '>=%' THEN SUBSTR(d.version_constraint, 3)
           WHEN d.version_constraint LIKE '<=%' THEN SUBSTR(d.version_constraint, 3)
           WHEN d.version_constraint LIKE '%@%' THEN SUBSTR(d.version_constraint, INSTR(d.version_constraint, '@') + 1)
           ELSE d.version_constraint
       END as constraint_version
   FROM build_dependencies d
   WHERE d.package_id = ?
),
version_parts AS (
   SELECT
       p.name,
       p.id,
       p.version,
       p.serial,
       p.location,
       CAST(SUBSTR(p.version, 1, INSTR(p.version || '.', '.') - 1) AS INTEGER) as major,
       CASE
           WHEN INSTR(p.version, '.') > 0 THEN
               CAST(SUBSTR(
                   SUBSTR(p.version, INSTR(p.version, '.') + 1),
                   1,
                   INSTR(SUBSTR(p.version, INSTR(p.version, '.') + 1) || '.', '.') - 1
               ) AS INTEGER)
           ELSE 0
       END as minor,
       CASE
           WHEN (LENGTH(p.version) - LENGTH(REPLACE(p.version, '.', ''))) >= 2 THEN
               CAST(SUBSTR(
                   p.version,
                   INSTR(p.version, '.') + INSTR(SUBSTR(p.version, INSTR(p.version, '.') + 1), '.') + 1
               ) AS INTEGER)
           ELSE 0
       END as patch
   FROM packages p
),
constraint_version_parts AS (
   SELECT
       di.dependency_name,
       di.version_constraint,
       di.location,
       di.constraint_type,
       di.constraint_version,
       CAST(SUBSTR(di.constraint_version, 1, INSTR(di.constraint_version || '.', '.') - 1) AS INTEGER) as const_major,
       CASE
           WHEN INSTR(di.constraint_version, '.') > 0 THEN
               CAST(SUBSTR(
                   SUBSTR(di.constraint_version, INSTR(di.constraint_version, '.') + 1),
                   1,
                   INSTR(SUBSTR(di.constraint_version, INSTR(di.constraint_version, '.') + 1) || '.', '.') - 1
               ) AS INTEGER)
           ELSE 0
       END as const_minor,
       CASE
           WHEN (LENGTH(di.constraint_version) - LENGTH(REPLACE(di.constraint_version, '.', ''))) >= 2 THEN
               CAST(SUBSTR(
                   di.constraint_version,
                   INSTR(di.constraint_version, '.') + INSTR(SUBSTR(di.constraint_version, INSTR(di.constraint_version, '.') + 1), '.') + 1
               ) AS INTEGER)
           ELSE 0
       END as const_patch
   FROM dependency_info di
   WHERE di.constraint_version IS NOT NULL
),
matched_packages AS (
   SELECT
       di.dependency_name,
       di.version_constraint,
       di.location,
       di.constraint_type,
       vp.id as package_id,
       vp.version as package_version,
       vp.serial,
       vp.major,
       vp.minor,
       vp.patch,
       RANK() OVER (
           PARTITION BY di.dependency_name, di.location
           ORDER BY vp.major DESC, vp.minor DESC, vp.patch DESC
       ) as version_rank
   FROM dependency_info di
   JOIN version_parts vp ON vp.name = di.dependency_name AND vp.location = di.location
   LEFT JOIN constraint_version_parts cvp ON di.dependency_name = cvp.dependency_name
       AND di.location = cvp.location
       AND di.constraint_type = cvp.constraint_type
   WHERE
       (di.constraint_type = 'HIGHEST')
       OR (di.constraint_type = 'EXACT' AND vp.version = di.constraint_version)
       OR (di.constraint_type = 'GREATER_EQUAL' AND (
           vp.major > cvp.const_major OR
           (vp.major = cvp.const_major AND vp.minor > cvp.const_minor) OR
           (vp.major = cvp.const_major AND vp.minor = cvp.const_minor AND vp.patch >= cvp.const_patch)
       ))
       OR (di.constraint_type = 'LESS_EQUAL' AND (
           vp.major < cvp.const_major OR
           (vp.major = cvp.const_major AND vp.minor < cvp.const_minor) OR
           (vp.major = cvp.const_major AND vp.minor = cvp.const_minor AND vp.patch <= cvp.const_patch)
       ))
)
SELECT
   package_id,
   location,
   package_version,
   serial
FROM matched_packages
WHERE version_rank = 1;`

	ConflictsQuery = `WITH dependency_info AS (
   SELECT
       d.dependency_name,
       d.version_constraint,
       d.location,
       CASE
           WHEN d.version_constraint = char(0) THEN 'HIGHEST'
           WHEN d.version_constraint LIKE '>=%' THEN 'GREATER_EQUAL'
           WHEN d.version_constraint LIKE '<=%' THEN 'LESS_EQUAL'
           WHEN d.version_constraint LIKE '%@%' THEN 'EXACT'
           ELSE 'EXACT'
       END as constraint_type,
       CASE
           WHEN d.version_constraint = char(0) THEN NULL
           WHEN d.version_constraint LIKE '>=%' THEN SUBSTR(d.version_constraint, 3)
           WHEN d.version_constraint LIKE '<=%' THEN SUBSTR(d.version_constraint, 3)
           WHEN d.version_constraint LIKE '%@%' THEN SUBSTR(d.version_constraint, INSTR(d.version_constraint, '@') + 1)
           ELSE d.version_constraint
       END as constraint_version
   FROM conflicts d
   WHERE d.package_id = ?
),
version_parts AS (
   SELECT
       p.name,
       p.id,
       p.version,
       p.serial,
       p.location,
       CAST(SUBSTR(p.version, 1, INSTR(p.version || '.', '.') - 1) AS INTEGER) as major,
       CASE
           WHEN INSTR(p.version, '.') > 0 THEN
               CAST(SUBSTR(
                   SUBSTR(p.version, INSTR(p.version, '.') + 1),
                   1,
                   INSTR(SUBSTR(p.version, INSTR(p.version, '.') + 1) || '.', '.') - 1
               ) AS INTEGER)
           ELSE 0
       END as minor,
       CASE
           WHEN (LENGTH(p.version) - LENGTH(REPLACE(p.version, '.', ''))) >= 2 THEN
               CAST(SUBSTR(
                   p.version,
                   INSTR(p.version, '.') + INSTR(SUBSTR(p.version, INSTR(p.version, '.') + 1), '.') + 1
               ) AS INTEGER)
           ELSE 0
       END as patch
   FROM packages p
),
constraint_version_parts AS (
   SELECT
       di.dependency_name,
       di.version_constraint,
       di.location,
       di.constraint_type,
       di.constraint_version,
       CAST(SUBSTR(di.constraint_version, 1, INSTR(di.constraint_version || '.', '.') - 1) AS INTEGER) as const_major,
       CASE
           WHEN INSTR(di.constraint_version, '.') > 0 THEN
               CAST(SUBSTR(
                   SUBSTR(di.constraint_version, INSTR(di.constraint_version, '.') + 1),
                   1,
                   INSTR(SUBSTR(di.constraint_version, INSTR(di.constraint_version, '.') + 1) || '.', '.') - 1
               ) AS INTEGER)
           ELSE 0
       END as const_minor,
       CASE
           WHEN (LENGTH(di.constraint_version) - LENGTH(REPLACE(di.constraint_version, '.', ''))) >= 2 THEN
               CAST(SUBSTR(
                   di.constraint_version,
                   INSTR(di.constraint_version, '.') + INSTR(SUBSTR(di.constraint_version, INSTR(di.constraint_version, '.') + 1), '.') + 1
               ) AS INTEGER)
           ELSE 0
       END as const_patch
   FROM dependency_info di
   WHERE di.constraint_version IS NOT NULL
),
matched_packages AS (
   SELECT
       di.dependency_name,
       di.version_constraint,
       di.location,
       di.constraint_type,
       vp.id as package_id,
       vp.version as package_version,
       vp.serial,
       vp.major,
       vp.minor,
       vp.patch,
       RANK() OVER (
           PARTITION BY di.dependency_name, di.location
           ORDER BY vp.major DESC, vp.minor DESC, vp.patch DESC
       ) as version_rank
   FROM dependency_info di
   JOIN version_parts vp ON vp.name = di.dependency_name AND vp.location = di.location
   LEFT JOIN constraint_version_parts cvp ON di.dependency_name = cvp.dependency_name
       AND di.location = cvp.location
       AND di.constraint_type = cvp.constraint_type
   WHERE
       (di.constraint_type = 'HIGHEST')
       OR (di.constraint_type = 'EXACT' AND vp.version = di.constraint_version)
       OR (di.constraint_type = 'GREATER_EQUAL' AND (
           vp.major > cvp.const_major OR
           (vp.major = cvp.const_major AND vp.minor > cvp.const_minor) OR
           (vp.major = cvp.const_major AND vp.minor = cvp.const_minor AND vp.patch >= cvp.const_patch)
       ))
       OR (di.constraint_type = 'LESS_EQUAL' AND (
           vp.major < cvp.const_major OR
           (vp.major = cvp.const_major AND vp.minor < cvp.const_minor) OR
           (vp.major = cvp.const_major AND vp.minor = cvp.const_minor AND vp.patch <= cvp.const_patch)
       ))
)
SELECT
   package_id,
   location,
   package_version,
   serial,
   dependency_name
FROM matched_packages
WHERE version_rank = 1;`

/*  can select
*     package_id,
*     dependency_name,
*     version_constraint,
*     location,
*     package_version,
*     serial
 */
)

func SolveDeps(id packet.PackageID, favoriteLocation string, installedDB *sql.DB, sourcesDB *sql.DB, maps *map[string]map[string]DependencyStatus) error {

	rows, err := sourcesDB.Query(RuntimeDependenciesQuery, id)
	if err != nil {
		return err
	}
	var packageId, location, version, packageName string
	var pkgSerial int

	tempMap := *maps
	// RUNTIME
	for rows.Next() {
		if err := rows.Scan(&packageId, &location, &version, &pkgSerial, &packageName); err != nil {
			rows.Close()
			return err
		}

		packageIdNormalized := packet.NewId(packageId)
		if err := SolveDeps(packageIdNormalized, favoriteLocation, installedDB, sourcesDB, &tempMap); err != nil {
			rows.Close()
			return err
		}

		installedID, installedSerial, installedLocation, err := database.CheckVersionInstalled(packageName, installedDB)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return err
			}
		}

		if (location == installedLocation && installedSerial > pkgSerial) || (installedID.Version() == packageIdNormalized.Version()) {
			continue
		}

		if v, exists := tempMap["runtime"][packageName]; exists {
			if location == favoriteLocation || (v.Location == location && v.Serial < pkgSerial) {
			} else {
				continue
			}
		}

		tempMap["runtime"][packageName] = DependencyStatus{
			Id:       packageIdNormalized,
			Serial:   pkgSerial,
			Location: location,
		}
	}
	rows.Close()

	// BUILD
	rows, err = sourcesDB.Query(BuildDependenciesQuery, id)
	if err != nil {
		return err
	}

	for rows.Next() {
		if err := rows.Scan(&packageId, &location, &version, &pkgSerial, &packageName); err != nil {
			rows.Close()
			return err
		}

		packageIdNormalized := packet.NewId(packageId)
		if err := SolveDeps(packageIdNormalized, favoriteLocation, installedDB, sourcesDB, &tempMap); err != nil {
			rows.Close()
			return err
		}

		installedID, installedSerial, installedLocation, err := database.CheckVersionInBuild(packageName, installedDB)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return err
			}
		}

		if (location == installedLocation && installedSerial > pkgSerial) || (installedID.Version() == packageIdNormalized.Version()) {
			continue
		}

		if v, exists := tempMap["build"][packageName]; exists {
			if location == favoriteLocation || (v.Location == location && v.Serial < pkgSerial) {
			} else {
				continue
			}
		}

		tempMap["build"][packageName] = DependencyStatus{
			Id:       packageIdNormalized,
			Serial:   pkgSerial,
			Location: location,
		}
	}
	rows.Close()

	// CONFLICTS
	rows, err = sourcesDB.Query(BuildDependenciesQuery, id)
	if err != nil {
		return err
	}

	for rows.Next() {
		if err := rows.Scan(&packageId, &location, &version, &pkgSerial, &packageName); err != nil {
			rows.Close()
			return err
		}

		packageIdNormalized := packet.NewId(packageId)
		if err := SolveDeps(packageIdNormalized, favoriteLocation, installedDB, sourcesDB, &tempMap); err != nil {
			rows.Close()
			return err
		}

		if v, exists := tempMap["conflicts"][packageName]; exists {
			if location == favoriteLocation || (v.Location == location && v.Serial > pkgSerial) {
			} else {
				continue
			}
		}

		tempMap["conflicts"][packageName] = DependencyStatus{
			Id:       packet.NewId(packageId),
			Serial:   pkgSerial,
			Location: location,
		}
	}
	rows.Close()

	return nil
}
