// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for German (`de`).
class AppLocalizationsDe extends AppLocalizations {
  AppLocalizationsDe([String locale = 'de']) : super(locale);

  @override
  String get about => 'Über';

  @override
  String get appearance => 'Darstellung';

  @override
  String get account => 'Konto';

  @override
  String get account_delete_confirm =>
      'Du bist dabei, dein Konto zu löschen. Alle deine Routen werden ebenfalls gelöscht. Möchtest du fortfahren?';

  @override
  String get account_privacy => 'Privatsphäre des Kontos';

  @override
  String get add_bio => 'Bio hinzufügen';

  @override
  String get add_waypoint => 'Wegpunkt hinzufügen';

  @override
  String get adjust_track => 'Pfad anpassen';

  @override
  String get after => 'Nach';

  @override
  String get all => 'Alle';

  @override
  String get altitude => 'Höhe';

  @override
  String get author => 'Autor';

  @override
  String get average_speed => 'Durchschn. Geschwindigkeit';

  @override
  String get background_location_body =>
      'wanderer erfasst Standortdaten im Hintergrund, damit deine Tour weiter aufgezeichnet wird, wenn der Bildschirm aus ist oder du die App schließt. Deine Aufzeichnung bleibt auf deinem Gerät, bis du die Tour speicherst.';

  @override
  String get background_location_confirm => 'Weiter';

  @override
  String get background_location_title =>
      'Aufzeichnung im Hintergrund fortsetzen';

  @override
  String get basic_info => 'Basisinformation';

  @override
  String get before => 'Vor';

  @override
  String get by => 'von';

  @override
  String get cancel => 'Abbrechen';

  @override
  String get discard => 'Verwerfen';

  @override
  String get discard_trail_confirm =>
      'Diese Route und ihre Änderungen verwerfen?';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Karten',
      one: 'Karte',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Kategorien';

  @override
  String get category => 'Kategorie';

  @override
  String get change_email => 'Email ändern';

  @override
  String get change_password => 'Passwort ändern';

  @override
  String get clear_all => 'Alle ausblenden';

  @override
  String get close => 'Schließen';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Kommentare',
      one: 'Kommentar',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Abgeschlossen';

  @override
  String get completion_status => 'Abschlussstatus';

  @override
  String get confirm_deletion => 'Löschen bestätigen';

  @override
  String get couldnt_start_navigation =>
      'Navigation konnte nicht gestartet werden. Prüfe deine Verbindung und versuche es erneut.';

  @override
  String get location_services_disabled =>
      'Ortungsdienste sind deaktiviert. Bitte aktiviere GPS, um die Navigation zu nutzen.';

  @override
  String get location_permission_denied =>
      'Für die Navigation wird eine Standortberechtigung benötigt.';

  @override
  String get location_permission_permanently_denied =>
      'Die Standortberechtigung wurde dauerhaft verweigert. Bitte aktiviere sie in den Einstellungen.';

  @override
  String get location_unavailable =>
      'Standort konnte nicht ermittelt werden. Bitte versuchen Sie es erneut.';

  @override
  String get copy_link => 'Link kopieren';

  @override
  String get create_waypoint => 'Wegpunkt erstellen';

  @override
  String get creation_date => 'Erstellungsdatum';

  @override
  String get current_password => 'Aktuelles Passwort';

  @override
  String get danger_zone => 'Gefahrenzone';

  @override
  String get date => 'Datum';

  @override
  String get delete => 'Löschen';

  @override
  String get not_now => 'Jetzt nicht';

  @override
  String get open => 'Öffnen';

  @override
  String get delete_account => 'Konto löschen';

  @override
  String get delete_trail_confirm =>
      'Möchtest Du diese Route wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.';

  @override
  String get delete_blocked_while_uploading =>
      'This trail is uploading right now. Wait for the upload to finish, then try again.';

  @override
  String get delete_unsynced_trail_confirm =>
      'Delete this trail? It hasn\'t been uploaded yet, so this can\'t be undone.';

  @override
  String get delete_needs_connection =>
      'This trail is already on the server. Connect to the internet to delete it.';

  @override
  String get description => 'Beschreibung';

  @override
  String get difficult => 'Schwer';

  @override
  String get difficulty => 'Schwierigkeit';

  @override
  String get directions => 'Wegbeschreibung';

  @override
  String get distance => 'Distanz';

  @override
  String get download => 'Herunterladen';

  @override
  String get duration => 'Dauer';

  @override
  String get easy => 'Einfach';

  @override
  String get edit => 'Bearbeiten';

  @override
  String get edit_needs_connection =>
      'Editing works on the server copy of this trail. Connect to the internet to edit it.';

  @override
  String get edit_waypoint => 'Wegpunkt bearbeiten';

  @override
  String get edited => 'bearbeitet';

  @override
  String get elevation_gain => 'Höhenunterschied (aufw.)';

  @override
  String get elevation_loss => 'Höhenunterschied (abw.)';

  @override
  String get elevation_profile => 'Höhenprofil';

  @override
  String get email => 'E-Mail';

  @override
  String get email_not_unique => 'Diese Email_Adresse wird bereits verwendet';

  @override
  String get email_updated => 'E_Mail aktualisiert';

  @override
  String get error_deleting_trail => 'Error deleting trail';

  @override
  String get error_reading_file => 'Fehler beim Lesen der Datei';

  @override
  String get error_saving_settings => 'Fehler beim Speichern der Einstellungen';

  @override
  String get error_saving_trail => 'Fehler beim Speichern der Route';

  @override
  String get error_updating_password =>
      'Fehler beim Aktualisieren des Passworts';

  @override
  String get explore => 'Erkunden';

  @override
  String get exit_navigation => 'Beenden';

  @override
  String get stop_navigation_confirm =>
      'Navigation beenden und zur Route zurückkehren?';

  @override
  String get stop_recording => 'Aufnahme stoppen';

  @override
  String get stop_recording_confirm => 'Aufnahme stoppen?';

  @override
  String get search_this_area => 'In diesem Bereich suchen';

  @override
  String get filter_tags => 'Tags filtern';

  @override
  String get filter_trails => 'Routen filtern';

  @override
  String get finish => 'Ziel';

  @override
  String get finish_disabled_hint =>
      'Fügen Sie mindestens 2 Ankerpunkte hinzu, um Ihre Route abzuschließen.';

  @override
  String get follow => 'Folgen';

  @override
  String get followers => 'Follower';

  @override
  String get following => 'Folgt';

  @override
  String get from_photos => 'Aus Fotos';

  @override
  String get help => 'Hilfe';

  @override
  String get hiking => 'Wandern';

  @override
  String get home => 'Zuhause';

  @override
  String get icon => 'Icon';

  @override
  String get imperial => 'Imperial';

  @override
  String get joined => 'Beigetreten';

  @override
  String get language => 'Sprache';

  @override
  String get latitude => 'Breitengrad';

  @override
  String get like_status => '\"Gefällt mir\" Status';

  @override
  String get liked => 'Gefällt mir';

  @override
  String get likes => 'Likes';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Listen',
      one: 'Liste',
    );
    return '$_temp0';
  }

  @override
  String get location => 'Standort';

  @override
  String get locations => 'Orte';

  @override
  String get center_on_my_location => 'Auf meinen Standort zentrieren';

  @override
  String get location_tracking_notification_title => 'wanderer';

  @override
  String get location_tracking_notification_text => 'Route wird aufgezeichnet';

  @override
  String location_tracking_notification_text_navigating(String trail) {
    return 'Navigation: $trail';
  }

  @override
  String get login => 'Login';

  @override
  String get logout => 'Logout';

  @override
  String get longitude => 'Längengrad';

  @override
  String get map => 'Karte';

  @override
  String get metric => 'Metrisch';

  @override
  String get moderate => 'Mittel';

  @override
  String get my_account => 'Mein Konto';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String get name => 'Name';

  @override
  String get navigate => 'Navigieren';

  @override
  String get new_password => 'Neues Passwort';

  @override
  String get new_password_success => 'Neues Passwort wurde gesetzt';

  @override
  String get new_trail => 'Neue Route';

  @override
  String get trail_source_planner => 'Routen-Planer öffnen';

  @override
  String get trail_source_planner_description =>
      'Zeichne eine neue Route auf der Karte, Wegpunkt für Wegpunkt.';

  @override
  String get trail_source_record => 'Route aufzeichnen';

  @override
  String get trail_source_record_description =>
      'Verfolge deine Position live und zeichne deine Tour in Echtzeit auf.';

  @override
  String get trail_source_import => 'Datei importieren';

  @override
  String get trail_source_import_description =>
      'Lade GPX-, KML-, KMZ-, TCX- oder FIT-Dateien direkt von deinem Gerät hoch.';

  @override
  String get trail_source_import_error =>
      'Datei konnte nicht importiert werden';

  @override
  String get trail_source_offline_import_error =>
      'Offline können nur GPX-Dateien importiert werden';

  @override
  String get no_comments_so_far => 'Bisher keine Kommentare';

  @override
  String get no_data => 'Keine Daten';

  @override
  String get no_description_for_now => 'Noch keine Beschreibung';

  @override
  String get no_gps_data_in_image => 'Keine GPS Daten im Bild';

  @override
  String get no_preference => 'Keine Präferenz';

  @override
  String get no_trails_found => 'Keine Routen gefunden';

  @override
  String get not_completed => 'Nicht abgeschlossen';

  @override
  String get notifications => 'Benachrichtigungen';

  @override
  String get only_me => 'Nur ich';

  @override
  String get or => 'oder';

  @override
  String get own_trails_empty_body =>
      'Trails you record or save offline appear here, and upload automatically once you\'re back online.';

  @override
  String get own_trails_empty_title => 'Nothing saved yet';

  @override
  String get own_trails_offline_banner =>
      'Offline — showing only trails on this device.';

  @override
  String get trails_on_device => 'Trails (on device)';

  @override
  String get pause => 'Pausieren';

  @override
  String get password => 'Passwort';

  @override
  String get password_confirm => 'Passwort bestätigen';

  @override
  String get passwords_must_match => 'Passwörter müssen übereinstimmen';

  @override
  String photo_copy_failed_toast(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Trail saved, but $count photos couldn\'t be saved.',
      one: 'Trail saved, but 1 photo couldn\'t be saved.',
    );
    return '$_temp0';
  }

  @override
  String get photos => 'Fotos';

  @override
  String photos_skipped_no_gps(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count Fotos übersprungen — keine GPS-Daten',
      one: '1 Foto übersprungen — keine GPS-Daten',
    );
    return '$_temp0';
  }

  @override
  String get privacy => 'Privatsphäre';

  @override
  String get private => 'Privat';

  @override
  String get profile => 'Profil';

  @override
  String get public => 'Öffentlich';

  @override
  String get reached_end_of_trail => 'Du hast das Ende des Wegs erreicht.';

  @override
  String get register => 'Registrieren';

  @override
  String get reorder_photos_hint =>
      'Lange drücken und ziehen, um Fotos neu zu ordnen.';

  @override
  String get reset => 'Zurücksetzen';

  @override
  String get resume => 'Fortsetzen';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Navigation von $trail fortsetzen?';
  }

  @override
  String get resume_recording_prompt => 'Aufnahme fortsetzen?';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routen',
      one: 'Route',
    );
    return '$_temp0';
  }

  @override
  String get follow_roads => 'Straßen folgen';

  @override
  String get follow_roads_description =>
      'Passen Sie die aufgezeichnete Route an die nächstgelegenen Straßen und Wege an.';

  @override
  String get recalculate_heights => 'Höhen neu berechnen';

  @override
  String get recalculate_heights_description =>
      'Ersetzen Sie die aufgezeichnete GPS-Höhe durch genauere Werte aus Kartendaten.';

  @override
  String get save => 'Speichern';

  @override
  String get save_recording_options => 'Aufnahme speichern';

  @override
  String get save_track => 'Pfad speichern';

  @override
  String get search => 'Suche';

  @override
  String get search_for_trails_places => 'Suche nach Routen, Listen, Orten';

  @override
  String get select_date => 'Datum auswählen';

  @override
  String get settings => 'Einstellungen';

  @override
  String get settings_notification_comment_mention =>
      'Jemand hat dich in einem Kommentar erwähnt';

  @override
  String get settings_notification_list_share =>
      'Jemand hat eine Liste mit Dir geteilt';

  @override
  String get settings_notification_new_follower =>
      'Du hast einen neuen Follower';

  @override
  String get settings_notification_summit_log_create =>
      'Jemand hat einen Gipfelbuch_Eintrag zu deiner Route erstellt';

  @override
  String get settings_notification_summit_log_mention =>
      'Jemand hat dich in einem Gipfelbuch Eintrag erwähnt';

  @override
  String get settings_notification_trail_comment =>
      'Jemand hat einen Kommentar zu Deiner Route hinterlassen';

  @override
  String get settings_notification_trail_like =>
      'Jemand hat deiner Route ein \"Gefällt mir\" gegeben';

  @override
  String get settings_notification_trail_mention =>
      'Jemand hat dich in einer Route erwähnt';

  @override
  String get settings_notification_trail_share =>
      'Jemand hat eine Route mit Dir geteilt';

  @override
  String get settings_privacy_account_private =>
      'Nur Du kannst dein Profil sehen. Du wirst nicht in den Suchergebnissen angezeigt. Andere Benutzer können Dir nicht folgen oder Routen mit Dir teilen. Du kannst weiterhin Routen oder Listen veröffentlichen.';

  @override
  String get settings_privacy_account_public =>
      'Jeder kann Dein Profil sehen. Du erscheinst in den Suchergebnissen. Andere Benutzer können Dir folgen und Routen mit Dir teilen.';

  @override
  String get settings_privacy_lists_private =>
      'Deine Listen sind standardmäßig privat. Niemand außer Dir kann sie sehen. Du kannst diese Einstellung jederzeit für einzelne Listen ändern.';

  @override
  String get settings_privacy_lists_public =>
      'Deine Listen sind standardmäßig öffentlich. Jeder kann sie sehen. Du kannst diese Einstellung jederzeit für einzelne Listen ändern.';

  @override
  String get settings_privacy_trails_private =>
      'Deine Routen sind standardmäßig privat. Niemand außer Dir kann sie sehen. Du kannst diese Einstellung jederzeit für einzelne Routen ändern.';

  @override
  String get settings_privacy_trails_public =>
      'Deine Trails sind standardmäßig öffentlich. Jeder kann sie sehen. Du kannst diese Einstellung jederzeit für einzelne Routen ändern.';

  @override
  String get share => 'Teilen';

  @override
  String get share_profile => 'Profil teilen';

  @override
  String get shared => 'Geteilt';

  @override
  String get show_on_map => 'Auf der Karte anzeigen';

  @override
  String signout_unsynced_warning(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'You have $count trails not uploaded yet. Signing out won\'t delete them — they\'ll be here when you sign back in — but they won\'t upload until then.',
      one:
          'You have 1 trail not uploaded yet. Signing out won\'t delete it — it\'ll be here when you sign back in — but it won\'t upload until then.',
    );
    return '$_temp0';
  }

  @override
  String get slogan => 'Deine Routen. Deine Daten. Dein Server.';

  @override
  String get sort => 'Sortieren';

  @override
  String get speed => 'Geschwindigkeit';

  @override
  String get start => 'Start';

  @override
  String get subcategories => 'Unterkategorien';

  @override
  String get summit_book => 'Gipfelbuch';

  @override
  String get sync_failed => 'Upload failed · Tap to retry';

  @override
  String get sync_pending => 'Waiting to upload';

  @override
  String get sync_uploading => 'Uploading…';

  @override
  String get tags => 'Tags';

  @override
  String get theme_dark => 'Dunkel';

  @override
  String get theme_light => 'Hell';

  @override
  String get theme_system => 'System folgen';

  @override
  String get time_in_motion => 'Zeit in Bewegung';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routen',
      one: 'Route',
    );
    return '$_temp0';
  }

  @override
  String get trail_saved_successfully => 'Route gespeichert';

  @override
  String get trail_not_on_this_device =>
      'This trail is no longer on this device.';

  @override
  String get trail_uploaded_reopen_to_edit =>
      'This trail finished uploading. Re-open it from your trails to keep editing.';

  @override
  String get some_waypoints_failed_to_save =>
      'Route gespeichert, aber einige Wegpunkte konnten nicht gespeichert werden';

  @override
  String get units => 'Einheiten';

  @override
  String get users => 'Benutzer';

  @override
  String get username => 'Nutzername';

  @override
  String get username_not_unique =>
      'Dieser Nutzername ist bereits vergeben. Bitte versuche es mit einem anderen.';

  @override
  String get visibilty_status => 'Sichtbarkeit';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Wegpunkte',
      one: 'Wegpunkt',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Willkommen bei';

  @override
  String get wrong_username_or_password =>
      'Falscher Nutzername oder falsches Passwort';

  @override
  String get you_have_arrived => 'Angekommen';

  @override
  String get settings_categories_confirm_disable_title =>
      'Diese Kategorie trotzdem ausblenden?';

  @override
  String get settings_categories_confirm_disable_subcategory_title =>
      'Diese Unterkategorie trotzdem ausblenden?';

  @override
  String settings_categories_confirm_disable_body(int count) {
    return '$count Ihrer Trails verwenden diese Kategorie. Sie bleiben veröffentlicht, aber dieser Filter wird nicht angezeigt.';
  }

  @override
  String get settings_categories_confirm_view_trails => 'Routen anzeigen';

  @override
  String get settings_categories_confirm_disable_confirm =>
      'Trotzdem deaktivieren';

  @override
  String get settings_categories_empty_title => 'Keine Unter-Kategorien';

  @override
  String get settings_categories_empty_body =>
      'Diese Kategorie hat keine zu konfigurierenden Unterkategorien.';

  @override
  String get settings_categories_reorder_hint =>
      'Kategorien bestimmen, welche Routentypen Sie sehen und in welcher Reihenfolge. Schalten Sie eine aus, um sie in Filtern zu verstecken — Ihre Routen bleiben veröffentlicht, sie erscheinen einfach nicht unter dieser Kategorie. Tippen Sie auf eine Kategorie, um ihre Unterkategorien individuell zu verwalten.\n\nUm die Reihenfolge zu ändern, drücken und halten Sie eine Zeile und ziehen Sie sie dann an eine neue Position. Die Reihenfolge, die Sie hier einstellen, wird überall verwendet, wo Kategorien angezeigt werden.';

  @override
  String get something_went_wrong => 'Etwas ist schief gelaufen';

  @override
  String get technical_details => 'Technische Details';

  @override
  String get link => 'Verknüpfen';

  @override
  String get url => 'URL';

  @override
  String get open_in_new_tab => 'In neuem Tab öffnen';

  @override
  String get remove => 'Entfernen';

  @override
  String get remove_download_confirm_body =>
      'This removes the downloaded copy from this device. The trail itself is not deleted — you\'ll need to download it again to use it offline.';

  @override
  String get apply => 'Anwenden';

  @override
  String get add_at_least_2_anchors_hint =>
      'Fügen Sie mindestens 2 Ankerpunkte hinzu, um das Höhenprofil zu sehen.';

  @override
  String get reverse_direction => 'Richtung umkehren';

  @override
  String get delete_all => 'Alle löschen';

  @override
  String get auto_routing => 'Auto-Routing aktivieren';

  @override
  String get auto_routing_hint =>
      'Automatisch den Straßen und Pfaden zwischen den Ankerpunkten folgen.';

  @override
  String get travel_profile => 'Fortbewegungsmittel';

  @override
  String get no_track_data => 'Keine Pfad-Daten';

  @override
  String get available_offline => 'Offline verfügbar';

  @override
  String get no_lists_found => 'Keine Listen gefunden';

  @override
  String get search_lists => 'Liste suchen…';

  @override
  String get search_for_a_location => 'Nach einem Ort suchen';

  @override
  String no_results_for_query(String query) {
    return 'Keine Ergebnisse für „$query“';
  }

  @override
  String get filter => 'Filter';

  @override
  String no_label_yet(String label) {
    return 'Noch kein $label.';
  }

  @override
  String get no_lists_yet => 'Noch keine Listen vorhanden.';

  @override
  String get no_bio_yet => 'Noch keine Biografie.';

  @override
  String get show_more => 'Mehr anzeigen';

  @override
  String get show_less => 'Weniger anzeigen';

  @override
  String get feed => 'Feed';

  @override
  String get no_trails_yet => 'Noch keine Routen.';

  @override
  String get library_empty_title => 'Keine heruntergeladenen Routen';

  @override
  String get library_empty_body =>
      'Routen, die du herunterlädst, werden hier gespeichert, damit du sie offline öffnen kannst.';

  @override
  String get library_empty_search_body =>
      'Versuche einen anderen Suchbegriff oder setze die Filter zurück.';

  @override
  String get search_location => 'Ort suchen';

  @override
  String no_servers_match_query(String query) {
    return 'Keine Server stimmen überein mit \"$query\"';
  }

  @override
  String get use_custom_url_instead => 'Eigene URL verwenden';

  @override
  String get select_instance => 'Instanz auswählen';

  @override
  String get enter_server_url_hint => 'Server-URL eingeben (z.B. wanderer.to)';

  @override
  String get search_library => 'Bibliothek durchsuchen…';

  @override
  String language_and_units(String language, String units) {
    return '$language & $units';
  }

  @override
  String get edit_route => 'Route bearbeiten';

  @override
  String get undo => 'Rückgängig';

  @override
  String get redo => 'Wiederholen';

  @override
  String get bold => 'Fett';

  @override
  String get italic => 'Kursiv';

  @override
  String get underline => 'Unterstrichen';

  @override
  String get bullet_list => 'Aufzählung';

  @override
  String get ordered_list => 'Geordnete Liste';

  @override
  String get blockquote => 'Zitat';

  @override
  String get library => 'Bibliothek';

  @override
  String get settings_offline_regions_title => 'Offline-Karten / Regionen';

  @override
  String get regions_search_hint => 'Nach Region suchen';

  @override
  String get regions_dem_toggle_label => 'Höhendaten herunterladen (DEM)';

  @override
  String get regions_dem_toggle_caption =>
      'Fügt Hangschattierung hinzu; vergrößert die Downloadgröße';

  @override
  String get regions_update_available => 'Update verfügbar';

  @override
  String get regions_update_action => 'Aktualisieren';

  @override
  String get regions_retry => 'Erneut versuchen';

  @override
  String get regions_not_yet_available => 'Noch nicht verfügbar';

  @override
  String get regions_build_failed => 'Build fehlgeschlagen';

  @override
  String regions_delete_confirm_title(String name) {
    return '$name löschen?';
  }

  @override
  String get regions_delete_confirm_body =>
      'Dies entfernt die heruntergeladene Karte und die Höhendaten für diese Region. Sie müssen sie erneut herunterladen, um sie offline zu nutzen.';

  @override
  String get regions_delete_confirm_action => 'Löschen';

  @override
  String regions_disk_usage_summary(String size, num count) {
    return '$size in $count heruntergeladenen Region(en)';
  }

  @override
  String get regions_empty_search_title => 'Keine passenden Regionen';

  @override
  String get regions_empty_search_body =>
      'Versuchen Sie es mit einem anderen Suchbegriff.';

  @override
  String get regions_empty_catalog_title => 'Keine Offline-Regionen verfügbar';

  @override
  String get regions_empty_catalog_body =>
      'Fragen Sie den Administrator Ihrer wanderer Instanz, die herunterladbaren Regionen zu konfigurieren.';

  @override
  String get regions_vector_tile_title => 'Vektor';

  @override
  String get regions_dem_tile_title => 'Höhenandaten';

  @override
  String get regions_download_failed => 'Download fehlgeschlagen';

  @override
  String get regions_dem_locked_subtitle => 'Zuerst Kartendaten herunterladen';

  @override
  String get regions_offline_unavailable_title =>
      'Regionen können nicht geladen werden';

  @override
  String get regions_offline_unavailable_body =>
      'Verbinden Sie sich mit dem Internet, um herunterladbare Regionen zu durchsuchen und zu verwalten.';

  @override
  String get regions_map_geometry_failed => 'Could not load region outline';

  @override
  String get regions_map_back_label => 'Back to regions';

  @override
  String regions_group_expand_label(String name) {
    return '$name ausklappen';
  }

  @override
  String regions_group_collapse_label(String name) {
    return '$name einklappen';
  }

  @override
  String get offline_title => 'Sie sind Offline';

  @override
  String get offline_try_again => 'Erneut versuchen';

  @override
  String get offline_map_body =>
      'Verbinde dich mit dem Internet, um die Karte zu laden. Heruntergeladene Routen sind verfügbar.';

  @override
  String get offline_list_body =>
      'Verbinden Sie sich mit dem Internet, um Listen zu laden.';

  @override
  String get offline_profile_body =>
      'Verbinden Sie sich mit dem Internet, um Ihr vollständiges Profil zu laden.';

  @override
  String get offline_settings_banner =>
      'Sie sind offline. Einstellungen sind schreibgeschützt, bis Sie sich wieder verbinden.';

  @override
  String get offline_action_unavailable =>
      'Du bist offline — versuche es erneut, sobald du wieder online bist.';

  @override
  String get offline_categories_body =>
      'Verbinden Sie sich mit dem Internet, um Kategorien zu verwalten.';

  @override
  String get offline_trail_search_body =>
      'Verbinde dich mit dem Internet, um nach Routen zu suchen. Heruntergeladene Routen sind verfügbar.';
}
