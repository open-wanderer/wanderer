// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Czech (`cs`).
class AppLocalizationsCs extends AppLocalizations {
  AppLocalizationsCs([String locale = 'cs']) : super(locale);

  @override
  String get about => 'O aplikaci';

  @override
  String get appearance => 'Vzhled';

  @override
  String get account => 'Účet';

  @override
  String get account_delete_confirm =>
      'Chystáte se smazat svůj účet. Smazány budou i všechny vaše trasy. Chcete pokračovat?';

  @override
  String get account_privacy => 'Ochrana osobních údajů';

  @override
  String get add_bio => 'Přidat informace o mně';

  @override
  String get add_waypoint => 'Přidat bod trasy';

  @override
  String get adjust_track => 'Upravit záznam trasy';

  @override
  String get after => 'Po';

  @override
  String get all => 'Vše';

  @override
  String get altitude => 'Nadmořská výška';

  @override
  String get author => 'Autor';

  @override
  String get average_speed => 'Průměrná rychlost';

  @override
  String get background_location_body =>
      'wanderer collects location data in the background so your trail keeps recording when the screen is off or the app is closed. Your recorded track stays on your device until you choose to save the trail.\n\nAndroid only offers this in system settings: open Location and choose \"Allow all the time\".';

  @override
  String get background_location_confirm => 'Open settings';

  @override
  String get background_location_title => 'Keep recording in the background';

  @override
  String get basic_info => 'Základní informace';

  @override
  String get before => 'Před';

  @override
  String get by => 'od';

  @override
  String get cancel => 'Zrušit';

  @override
  String get discard => 'Zahodit';

  @override
  String get discard_trail_confirm => 'Zahodit tuto trasu a její změny?';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Karet',
      few: 'Karty',
      one: 'Karta',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Kategorie';

  @override
  String get category => 'Kategorie';

  @override
  String get change_email => 'Změnit e-mail';

  @override
  String get change_password => 'Změnit heslo';

  @override
  String get clear_all => 'Vymazat vše';

  @override
  String get close => 'Zavřít';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Komentářů',
      few: 'Komentáře',
      one: 'Komentář',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Dokončeno';

  @override
  String get completion_status => 'Stav dokončení';

  @override
  String get confirm_deletion => 'Potvrdit odstranění';

  @override
  String get couldnt_start_navigation =>
      'Navigaci se nepodařilo spustit. Zkontrolujte připojení a zkuste to znovu.';

  @override
  String get location_services_disabled =>
      'Polohové služby jsou vypnuté. Chcete-li používat navigaci, zapněte GPS.';

  @override
  String get location_permission_denied =>
      'Navigace vyžaduje oprávnění k poloze.';

  @override
  String get location_permission_permanently_denied =>
      'Přístup k poloze je trvale zamítnut. Povolte jej v nastavení.';

  @override
  String get location_unavailable =>
      'Vaši polohu se nepodařilo určit. Zkuste to znovu.';

  @override
  String get copy_link => 'Kopírovat odkaz';

  @override
  String get create_waypoint => 'Vytvořit bod trasy';

  @override
  String get creation_date => 'Datum vytvoření';

  @override
  String get current_password => 'Současné heslo';

  @override
  String get danger_zone => 'Nebezpečná zóna';

  @override
  String get date => 'Datum';

  @override
  String get delete => 'Vymazat';

  @override
  String get not_now => 'Not now';

  @override
  String get open => 'Otevřít';

  @override
  String get delete_account => 'Odstranit účet';

  @override
  String get delete_trail_confirm =>
      'Doopravdy chcete tuto trasu smazat? Tuto akci nelze vrátit zpět.';

  @override
  String get delete_blocked_while_uploading =>
      'Tato trasa se právě nahrává. Počkejte na dokončení nahrávání a poté to zkuste znovu.';

  @override
  String get delete_unsynced_trail_confirm =>
      'Smazat tuto trasu? Ještě nebyla nahrána, takže tuto akci nelze vrátit zpět.';

  @override
  String get delete_needs_connection =>
      'Tato trasa již je na serveru. Chcete-li ji smazat, připojte se k internetu.';

  @override
  String get description => 'Popis';

  @override
  String get difficult => 'Náročné';

  @override
  String get difficulty => 'Obtížnost';

  @override
  String get directions => 'Navigovat';

  @override
  String get distance => 'Vzdálenost';

  @override
  String get download => 'Stáhnout';

  @override
  String get duration => 'Doba trvání';

  @override
  String get easy => 'Snadné';

  @override
  String get edit => 'Upravit';

  @override
  String get edit_needs_connection =>
      'Úpravy se provádějí na kopii této trasy uložené na serveru. Chcete-li ji upravit, připojte se k internetu.';

  @override
  String get edit_waypoint => 'Upravit bod trasy';

  @override
  String get edited => 'upraveno';

  @override
  String get elevation_gain => 'Stoupání';

  @override
  String get elevation_loss => 'Klesání';

  @override
  String get elevation_profile => 'Výškový profil';

  @override
  String get email => 'E-mail';

  @override
  String get email_not_unique => 'Tato e-mailová adresa je již obsazena.';

  @override
  String get email_updated => 'E-mail byl aktualizován';

  @override
  String get error_deleting_trail => 'Chyba při mazání trasy';

  @override
  String get error_reading_file => 'Soubor nelze načíst';

  @override
  String get error_saving_settings => 'Chyba při ukládání nastavení';

  @override
  String get error_saving_trail => 'Chyba při ukládání trasy';

  @override
  String get error_updating_password => 'Chyba během aktualizace hesla';

  @override
  String get explore => 'Objevování';

  @override
  String get exit_navigation => 'Ukončit';

  @override
  String get stop_navigation_confirm =>
      'Zastavit navigaci a vrátit se k trase?';

  @override
  String get stop_recording => 'Zastavit nahrávání';

  @override
  String get stop_recording_confirm => 'Zastavit nahrávání?';

  @override
  String get search_this_area => 'Hledat v této oblasti';

  @override
  String get filter_tags => 'Filtrovat podle štítků';

  @override
  String get filter_trails => 'Filtrovat trasy';

  @override
  String get finish => 'Konec';

  @override
  String get finish_disabled_hint =>
      'K dokončení trasy přidejte alespoň 2 body trasy.';

  @override
  String get follow => 'Sledovat';

  @override
  String get followers => 'Sledující';

  @override
  String get following => 'Sleduji';

  @override
  String get from_photos => 'Z fotografií';

  @override
  String get help => 'Nápověda';

  @override
  String get hiking => 'Turistika';

  @override
  String get home => 'Hlavní stránka';

  @override
  String get icon => 'Ikonka';

  @override
  String get imperial => 'Imperiální jednotky';

  @override
  String get joined => 'Připojeno';

  @override
  String get language => 'Jazyk';

  @override
  String get latitude => 'Zeměpisná šířka';

  @override
  String get like_status => 'Status „Líbí se mi“';

  @override
  String get liked => 'Líbilo se';

  @override
  String get likes => 'Líbí se';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Seznamů',
      few: 'Seznamy',
      one: 'Seznam',
    );
    return '$_temp0';
  }

  @override
  String get location => 'Poloha';

  @override
  String get locations => 'Místa';

  @override
  String get center_on_my_location => 'Vycentrovat na mou polohu';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Probíhá záznam trasy';

  @override
  String location_tracking_notification_text_navigating(String trail) {
    return 'Navigace po trase $trail';
  }

  @override
  String get login => 'Přihlásit se';

  @override
  String get logout => 'Odhlásit se';

  @override
  String get longitude => 'Zeměpisná délka';

  @override
  String get map => 'Mapa';

  @override
  String get metric => 'Metrické jednotky';

  @override
  String get moderate => 'Středně náročné';

  @override
  String get my_account => 'Můj účet';

  @override
  String in_distance(String distance) {
    return 'za $distance';
  }

  @override
  String get name => 'Název';

  @override
  String get navigate => 'Navigovat';

  @override
  String get new_password => 'Nové heslo';

  @override
  String get new_password_success => 'Nové heslo bylo nastaveno';

  @override
  String get new_trail => 'Nová trasa';

  @override
  String get trail_source_planner => 'Otevřít plánovač tras';

  @override
  String get trail_source_planner_description =>
      'Nakreslete na mapu novou trasu bod po bodu.';

  @override
  String get trail_source_record => 'Zaznamenat trasu';

  @override
  String get trail_source_record_description =>
      'Sledujte své aktuální souřadnice a zaznamenávejte cestu v reálném čase.';

  @override
  String get trail_source_import => 'Importovat soubor';

  @override
  String get trail_source_import_description =>
      'Nahrajte soubory GPX, KML, KMZ, TCX nebo FIT přímo z úložiště zařízení.';

  @override
  String get trail_source_import_error => 'Soubor se nepodařilo importovat';

  @override
  String get trail_source_offline_import_error =>
      'Offline lze importovat pouze soubory GPX';

  @override
  String get no_comments_so_far => 'Zatím žádné komentáře';

  @override
  String get no_data => 'Žádná data';

  @override
  String get no_description_for_now => 'Zatím bez popisu';

  @override
  String get no_gps_data_in_image => 'Obrázek neobsahuje GPS data';

  @override
  String get no_preference => 'Bez preference';

  @override
  String get no_trails_found => 'Nebyly nalezeny žádné trasy';

  @override
  String get not_completed => 'Nedokončeno';

  @override
  String get notifications => 'Oznámení';

  @override
  String get only_me => 'Jen já';

  @override
  String get or => 'nebo';

  @override
  String get own_trails_empty_body =>
      'Trasy, které zaznamenáte nebo uložíte offline, se zobrazí zde a automaticky se nahrají, jakmile budete znovu online.';

  @override
  String get own_trails_empty_title => 'Zatím nic uloženo';

  @override
  String get own_trails_offline_banner =>
      'Offline — zobrazují se pouze trasy v tomto zařízení.';

  @override
  String get trails_on_device => 'Trasy (v zařízení)';

  @override
  String get pause => 'Pozastavit';

  @override
  String get password => 'Heslo';

  @override
  String get password_confirm => 'Potvrďte heslo';

  @override
  String get passwords_must_match => 'Hesla se musí shodovat';

  @override
  String photo_copy_failed_toast(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Trasa byla uložena, ale $count fotografií se nepodařilo uložit.',
      few: 'Trasa byla uložena, ale $count fotografie se nepodařilo uložit.',
      one: 'Trasa byla uložena, ale 1 fotografii se nepodařilo uložit.',
    );
    return '$_temp0';
  }

  @override
  String get photos => 'Fotografie a videa';

  @override
  String photos_skipped_no_gps(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count fotografií přeskočeno — chybí data GPS',
      few: '$count fotografie přeskočeny — chybí data GPS',
      one: '1 fotografie přeskočena — chybí data GPS',
    );
    return '$_temp0';
  }

  @override
  String get privacy => 'Soukromí';

  @override
  String get private => 'Soukromé';

  @override
  String get profile => 'Profil';

  @override
  String get public => 'Veřejné';

  @override
  String get reached_end_of_trail => 'Dorazili jste na konec trasy.';

  @override
  String get register => 'Registrace';

  @override
  String get reorder_photos_hint =>
      'Dlouhým stisknutím a přetažením změníte pořadí fotografií.';

  @override
  String get reset => 'Obnovit';

  @override
  String get resume => 'Pokračovat';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Pokračovat v navigaci po trase $trail?';
  }

  @override
  String get resume_recording_prompt => 'Obnovit nahrávání?';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Tras',
      few: 'Trasy',
      one: 'Trasa',
    );
    return '$_temp0';
  }

  @override
  String get follow_roads => 'Vést po cestách';

  @override
  String get follow_roads_description =>
      'Přizpůsobit zaznamenanou trasu nejbližším silnicím a stezkám.';

  @override
  String get recalculate_heights => 'Přepočítat nadmořské výšky';

  @override
  String get recalculate_heights_description =>
      'Nahradit zaznamenané údaje o nadmořské výšce z GPS přesnějšími hodnotami z mapy.';

  @override
  String get save => 'Uložit';

  @override
  String get save_recording_options => 'Uložit záznam';

  @override
  String get save_track => 'Uložit trasu';

  @override
  String get search => 'Vyhledat';

  @override
  String get search_for_trails_places => 'Vyhledat trasy, seznamy, místa';

  @override
  String get select_date => 'Vybrat datum';

  @override
  String get settings => 'Nastavení';

  @override
  String get settings_notification_comment_mention =>
      'Někdo vás zmínil v komentáři';

  @override
  String get settings_notification_list_share => 'Někdo s vámi sdílel seznam';

  @override
  String get settings_notification_new_follower => 'Máte nového sledujícího';

  @override
  String get settings_notification_summit_log_create =>
      'Někdo přidal záznam o výstupu na vaší trase';

  @override
  String get settings_notification_summit_log_mention =>
      'Někdo vás zmínil v záznamu o výstupu';

  @override
  String get settings_notification_trail_comment =>
      'Někdo zanechal komentář k vaší trase';

  @override
  String get settings_notification_trail_like => 'Někomu se líbí vaše trasa';

  @override
  String get settings_notification_trail_mention => 'Někdo vás zmínil u trasy';

  @override
  String get settings_notification_trail_share => 'Někdo s vámi sdílel trasu';

  @override
  String get settings_privacy_account_private =>
      'Váš profil vidíte pouze vy. Nebudete se zobrazovat ve výsledcích vyhledávání. Ostatní uživatelé vás nemohou sledovat ani s vámi sdílet trasy. Stále však můžete trasy nebo seznamy zveřejňovat.';

  @override
  String get settings_privacy_account_public =>
      'Váš profil může vidět kdokoli. Budete se zobrazovat ve výsledcích vyhledávání. Ostatní uživatelé vás mohou sledovat a sdílet s vámi trasy.';

  @override
  String get settings_privacy_lists_private =>
      'Vaše seznamy jsou ve výchozím nastavení soukromé. Nikdo kromě vás je neuvidí. Toto nastavení můžete u jednotlivých seznamů kdykoli změnit.';

  @override
  String get settings_privacy_lists_public =>
      'Vaše seznamy jsou ve výchozím nastavení veřejné. Uvidí je kdokoli. Toto nastavení můžete u jednotlivých seznamů kdykoli změnit.';

  @override
  String get settings_privacy_trails_private =>
      'Vaše trasy jsou ve výchozím nastavení soukromé. Nikdo kromě vás je neuvidí. Toto nastavení můžete u jednotlivých tras kdykoli změnit.';

  @override
  String get settings_privacy_trails_public =>
      'Vaše trasy jsou ve výchozím nastavení veřejné. Uvidí je kdokoli. Toto nastavení můžete u jednotlivých tras kdykoli změnit.';

  @override
  String get share => 'Sdílet';

  @override
  String get share_profile => 'Sdílet profil';

  @override
  String get shared => 'Sdíleno';

  @override
  String get show_on_map => 'Zobrazit na mapě';

  @override
  String signout_unsynced_warning(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'Máte $count dosud nenahraných tras. Odhlášením je nesmažete — po opětovném přihlášení zde zůstanou — ale do té doby se nenahrají.',
      few:
          'Máte $count dosud nenahrané trasy. Odhlášením je nesmažete — po opětovném přihlášení zde zůstanou — ale do té doby se nenahrají.',
      one:
          'Máte 1 dosud nenahranou trasu. Odhlášením ji nesmažete — po opětovném přihlášení zde zůstane — ale do té doby se nenahraje.',
    );
    return '$_temp0';
  }

  @override
  String get slogan => 'Vaše trasy. Vaše data. Váš server.';

  @override
  String get sort => 'Seřadit';

  @override
  String get speed => 'Rychlost';

  @override
  String get start => 'Spustit';

  @override
  String get subcategories => 'Podkategorie';

  @override
  String get summit_book => 'Vrcholová kniha';

  @override
  String get sync_failed => 'Nahrávání selhalo · Klepnutím opakujete';

  @override
  String get sync_pending => 'Čeká na nahrání';

  @override
  String get sync_uploading => 'Nahrávání…';

  @override
  String get tags => 'Štítky';

  @override
  String get theme_dark => 'Tmavý';

  @override
  String get theme_light => 'Světlý';

  @override
  String get theme_system => 'Podle systému';

  @override
  String get time_in_motion => 'Čas v pohybu';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Tras',
      few: 'Trasy',
      one: 'Trasa',
    );
    return '$_temp0';
  }

  @override
  String get trail_saved_successfully => 'Trasa úspěšně uložena';

  @override
  String get trail_not_on_this_device =>
      'Tato trasa již není v tomto zařízení.';

  @override
  String get trail_uploaded_reopen_to_edit =>
      'Nahrávání této trasy bylo dokončeno. Chcete-li pokračovat v úpravách, znovu ji otevřete ze svých tras.';

  @override
  String get some_waypoints_failed_to_save =>
      'Trasa byla uložena, některé body trasy se však nepodařilo uložit';

  @override
  String get units => 'Jednotky';

  @override
  String get users => 'Uživatelé';

  @override
  String get username => 'Uživatelské jméno';

  @override
  String get username_not_unique =>
      'Toto uživatelské jméno je již obsazeno. Zkuste prosím jiné.';

  @override
  String get visibilty_status => 'Stav viditelnosti';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Bodů trasy',
      few: 'Body trasy',
      one: 'Bod trasy',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Vítejte v';

  @override
  String get wrong_username_or_password =>
      'Nesprávné uživatelské jméno nebo heslo';

  @override
  String get you_have_arrived => 'Dorazili jste';

  @override
  String get settings_categories_confirm_disable_title =>
      'Skrýt tuto kategorii?';

  @override
  String get settings_categories_confirm_disable_subcategory_title =>
      'Skrýt tuto podkategorii?';

  @override
  String settings_categories_confirm_disable_body(int count) {
    return 'Počet vašich tras používajících tuto kategorii: $count. Trasy zůstanou zveřejněné, ale tento filtr bude skrytý.';
  }

  @override
  String get settings_categories_confirm_view_trails => 'Zobrazit trasy';

  @override
  String get settings_categories_confirm_disable_confirm => 'Přesto skrýt';

  @override
  String get settings_categories_empty_title => 'Žádné podkategorie';

  @override
  String get settings_categories_empty_body =>
      'Tato kategorie nemá žádné podkategorie, které by bylo možné nastavit.';

  @override
  String get settings_categories_reorder_hint =>
      'Kategorie určují, které typy tras uvidíte a v jakém pořadí. Vypnutím kategorie ji skryjete ve filtrech — vaše trasy zůstanou zveřejněné, pouze se přestanou zobrazovat pod touto kategorií. Klepnutím na kategorii můžete samostatně spravovat její podkategorie.\n\nChcete-li změnit pořadí, dlouze stiskněte řádek a přetáhněte jej na nové místo. Zvolené pořadí se použije všude, kde se zobrazují kategorie.';

  @override
  String get something_went_wrong => 'Něco se pokazilo';

  @override
  String get technical_details => 'Technické podrobnosti';

  @override
  String get link => 'Propojit';

  @override
  String get url => 'Adresa URL';

  @override
  String get open_in_new_tab => 'Otevřít na nové kartě';

  @override
  String get remove => 'Odebrat';

  @override
  String get remove_download_confirm_body =>
      'Tímto odeberete staženou kopii z tohoto zařízení. Samotná trasa nebude smazána — pro použití offline ji budete muset stáhnout znovu.';

  @override
  String get apply => 'Použít';

  @override
  String get add_at_least_2_anchors_hint =>
      'K zobrazení výškového profilu přidejte alespoň 2 body trasy.';

  @override
  String get reverse_direction => 'Obrátit směr';

  @override
  String get delete_all => 'Vymazat vše';

  @override
  String get auto_routing => 'Automatické trasování';

  @override
  String get auto_routing_hint =>
      'Automaticky vést po silnicích a cestách mezi body trasy.';

  @override
  String get travel_profile => 'Způsob přesunu';

  @override
  String get no_track_data => 'Žádná data trasy';

  @override
  String get available_offline => 'Dostupné offline';

  @override
  String get no_lists_found => 'Nebyly nalezeny žádné seznamy';

  @override
  String get search_lists => 'Vyhledat v seznamech…';

  @override
  String get search_for_a_location => 'Vyhledat místo';

  @override
  String no_results_for_query(String query) {
    return 'Pro „$query“ nebyly nalezeny žádné výsledky';
  }

  @override
  String get filter => 'Filtrovat';

  @override
  String no_label_yet(String label) {
    return 'Zatím žádné $label.';
  }

  @override
  String get no_lists_yet => 'Zatím žádné seznamy.';

  @override
  String get no_bio_yet => 'Zatím bez informací o uživateli.';

  @override
  String get show_more => 'Zobrazit více';

  @override
  String get show_less => 'Zobrazit méně';

  @override
  String get feed => 'Kanál';

  @override
  String get no_trails_yet => 'Zatím žádné trasy.';

  @override
  String get library_empty_title => 'Žádné stažené trasy';

  @override
  String get library_empty_body =>
      'Stažené trasy jsou uloženy zde, abyste je mohli otevírat offline.';

  @override
  String get library_empty_search_body =>
      'Zkuste jiný hledaný výraz nebo vymažte filtry.';

  @override
  String get search_location => 'Vyhledat místo';

  @override
  String no_servers_match_query(String query) {
    return 'Výrazu „$query“ neodpovídají žádné servery';
  }

  @override
  String get use_custom_url_instead => 'Místo toho použít vlastní URL';

  @override
  String get select_instance => 'Vybrat instanci';

  @override
  String get enter_server_url_hint => 'Zadejte URL serveru (např. wanderer.to)';

  @override
  String get search_library => 'Prohledat knihovnu…';

  @override
  String language_and_units(String language, String units) {
    return '$language a $units';
  }

  @override
  String get edit_route => 'Upravit trasu';

  @override
  String get undo => 'Vrátit zpět';

  @override
  String get redo => 'Znovu';

  @override
  String get bold => 'Tučně';

  @override
  String get italic => 'Kurzíva';

  @override
  String get underline => 'Podtržení';

  @override
  String get bullet_list => 'Seznam s odrážkami';

  @override
  String get ordered_list => 'Číslovaný seznam';

  @override
  String get blockquote => 'Citace';

  @override
  String get library => 'Knihovna';

  @override
  String get settings_offline_regions_title => 'Offline mapy a oblasti';

  @override
  String get regions_search_hint => 'Vyhledat oblasti';

  @override
  String get regions_dem_toggle_label => 'Stáhnout výšková data (DEM)';

  @override
  String get regions_dem_toggle_caption =>
      'Přidá stínování reliéfu; zvětší velikost stahovaných dat';

  @override
  String get regions_update_available => 'Je dostupná aktualizace';

  @override
  String get regions_update_action => 'Aktualizovat';

  @override
  String get regions_retry => 'Opakovat';

  @override
  String get regions_not_yet_available => 'Zatím nedostupné';

  @override
  String get regions_build_failed => 'Příprava selhala';

  @override
  String regions_delete_confirm_title(String name) {
    return 'Smazat oblast $name?';
  }

  @override
  String get regions_delete_confirm_body =>
      'Tímto smažete staženou mapu a výšková data této oblasti. Pro použití offline je budete muset stáhnout znovu.';

  @override
  String get regions_delete_confirm_action => 'Vymazat';

  @override
  String regions_disk_usage_summary(String size, num count) {
    return 'Stažených oblastí: $count · využito $size';
  }

  @override
  String get regions_empty_search_title => 'Žádné odpovídající oblasti';

  @override
  String get regions_empty_search_body => 'Zkuste jiný hledaný výraz.';

  @override
  String get regions_empty_catalog_title =>
      'Nejsou dostupné žádné offline oblasti';

  @override
  String get regions_empty_catalog_body =>
      'Požádejte správce své instance Wandereru, aby nastavil oblasti ke stažení.';

  @override
  String get regions_vector_tile_title => 'Vektorová mapa';

  @override
  String get regions_dem_tile_title => 'Výšková data';

  @override
  String get regions_download_failed => 'Stahování selhalo';

  @override
  String get regions_dem_locked_subtitle => 'Nejprve stáhněte mapová data';

  @override
  String get regions_offline_unavailable_title => 'Oblasti nelze načíst';

  @override
  String get regions_offline_unavailable_body =>
      'Chcete-li procházet a spravovat oblasti ke stažení, připojte se k internetu.';

  @override
  String get regions_map_geometry_failed =>
      'Obrys oblasti se nepodařilo načíst';

  @override
  String get regions_map_back_label => 'Zpět na oblasti';

  @override
  String regions_group_expand_label(String name) {
    return 'Rozbalit $name';
  }

  @override
  String regions_group_collapse_label(String name) {
    return 'Sbalit $name';
  }

  @override
  String get offline_title => 'Jste offline';

  @override
  String get offline_try_again => 'Zkusit znovu';

  @override
  String get offline_map_body =>
      'Chcete-li načíst mapu, připojte se k internetu. Stažené trasy jsou nadále dostupné.';

  @override
  String get offline_list_body =>
      'Chcete-li načíst seznamy, připojte se k internetu.';

  @override
  String get offline_profile_body =>
      'Chcete-li načíst celý profil, připojte se k internetu.';

  @override
  String get offline_settings_banner =>
      'Jste offline. Nastavení je až do obnovení připojení pouze pro čtení.';

  @override
  String get offline_action_unavailable =>
      'Jste offline — zkuste to znovu, až budete opět online.';

  @override
  String get offline_categories_body =>
      'Chcete-li spravovat kategorie, připojte se k internetu.';

  @override
  String get offline_trail_search_body =>
      'Chcete-li vyhledávat trasy, připojte se k internetu. Stažené trasy jsou nadále dostupné.';
}
