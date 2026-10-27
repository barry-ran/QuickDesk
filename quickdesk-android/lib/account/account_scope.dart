import 'account_service.dart';

/// The UI-owned active account only; never create a second token/session owner.
/// Consumers capture both this instance and its epoch, not the mutable pointer.
class AccountScope {
  static AccountService? current;
}
